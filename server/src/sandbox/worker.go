package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"oj/server/parser"
	"oj/server/sandbox/resources"
	"oj/server/utility"
	"os"
	"path/filepath"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type TestStatus string

const (
	StatusPending TestStatus = "pending"
	StatusAC      TestStatus = "AC"
	StatusWA      TestStatus = "WA"
	StatusCE      TestStatus = "CE"
	StatusSE      TestStatus = "SE"
	StatusRE      TestStatus = "RE"
	StatusTLE     TestStatus = "TLE"
	StatusMLE     TestStatus = "MLE"
)

type CompilerOutput struct {
	ConfigLog  *string
	CompileLog *string
	ExitCode   int64
}

type WorkerOutput struct {
	Compiler    *CompilerOutput
	TestResults *parser.TestResults
}

type Worker struct {
	Id      string
	Moby    *client.Client
	Context context.Context
	Manager resources.SubmissionManager
}

func GetSubmissionPath(submissionId string) string {
	return filepath.Join(utility.EnvData.BasePath, "submissions", submissionId)
}

func (w Worker) Compile() (*CompilerOutput, error) {

	shellCommands := []string{
		"RUN apk add --no-cache cmake ninja-build",
	}

	config := container.Config{
		Image:           COMPILER_IMG_NAME,
		NetworkDisabled: true,
		Cmd: []string{
			"cmake", "-S", "src", "-B", "build", "-G", "Ninja", ">", "config.log", "2>&1", "&&",
			"cmake", "--build", "build", "--verbose", ">", "compile.log", "2>&1",
		},
		WorkingDir: "/workspace",
		Shell:      shellCommands,
	}

	hostConfig := container.HostConfig{
		Binds: []string{fmt.Sprintf("%s/submissions/%s:/workspace", utility.EnvData.BindBasePath, w.Id)},
		Tmpfs: map[string]string{
			"/workspace": "rw,noexec,nosuid,size=64m",
		},
		Resources: container.Resources{
			CPUCount: 1,
			Memory:   128 << 20,
		},
		NetworkMode: "none",
		Runtime:     "gvisor",
	}

	options := client.ContainerCreateOptions{
		Name:       "oj-compiler:" + w.Id,
		Config:     &config,
		HostConfig: &hostConfig,
	}

	res, err := w.setupContainerAndRun(options)

	basePath := w.Manager.GetBasePath()

	configLogBytes, err := os.ReadFile(filepath.Join(basePath, "config.log"))
	if err != nil {
		return nil, err
	}
	configLog := string(configLogBytes)

	compileLogBytes, err := os.ReadFile(filepath.Join(basePath, "compile.log"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return &CompilerOutput{ConfigLog: &configLog, ExitCode: res.StatusCode}, nil
		}
		return nil, err
	}
	compileLog := string(compileLogBytes)

	return &CompilerOutput{ConfigLog: &configLog, CompileLog: &compileLog, ExitCode: res.StatusCode}, nil
}

func (w Worker) Run(timeout int) (*parser.TestResults, error) {

	shellCommands := []string{
		"RUN apk add --no-cache libstdc++ cmake",
		"RUN adduser -D runner",
	}

	config := container.Config{
		Image:           COMPILER_IMG_NAME,
		NetworkDisabled: true,
		Cmd:             []string{"ctest", "--timeout", string(timeout), "--output-junit", "result.xml"},
		WorkingDir:      "/workspace",
		Shell:           shellCommands,
		User:            "runner",
	}

	hostConfig := container.HostConfig{
		Binds: []string{fmt.Sprintf("%s/submissions/%s/src/build:/workspace", utility.EnvData.BindBasePath, w.Id)},
		Tmpfs: map[string]string{
			"/workspace": "rw,noexec,nosuid,size=64m",
		},
		Resources: container.Resources{
			CPUCount: 1,
			Memory:   128 << 20,
		},
		NetworkMode: "none",
		Runtime:     "gvisor",
	}

	options := client.ContainerCreateOptions{
		Name:       "oj-runner:" + w.Id,
		Config:     &config,
		HostConfig: &hostConfig,
	}

	_, err := w.setupContainerAndRun(options)
	if err != nil {
		return nil, err
	}

	basePath := w.Manager.GetBasePath()
	resultBytes, err := os.ReadFile(filepath.Join(basePath, "src/build/result.xml"))
	if err != nil {
		return nil, err
	}
	return parser.ParseTestResults(resultBytes)
}

func (w Worker) setupContainerAndRun(options client.ContainerCreateOptions) (*container.WaitResponse, error) {
	createResult, err := w.Moby.ContainerCreate(w.Context, options)
	if err != nil {
		return nil, err
	}

	containerId := createResult.ID
	defer w.Moby.ContainerRemove(w.Context, containerId, client.ContainerRemoveOptions{RemoveVolumes: false})

	_, err = w.Moby.ContainerStart(w.Context, containerId, client.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	waitResult := w.Moby.ContainerWait(w.Context, containerId, client.ContainerWaitOptions{})
	res, err := <-waitResult.Result, <-waitResult.Error
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func CreateWorker(submissionId, problemId string, settings *parser.ProblemSettings, answer *parser.TestResults) (int, TestStatus, *WorkerOutput, error) {
	submissionManager := resources.SubmissionManager{Id: submissionId}
	defer submissionManager.ClearFiles()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	moby, err := client.New()
	if err != nil {
		return 0, StatusPending, nil, err
	}
	worker := Worker{
		Id:      submissionId,
		Moby:    moby,
		Context: ctx,
		Manager: submissionManager,
	}

	if err = submissionManager.CopyTestFiles(resources.ProblemManager{Id: problemId}); err != nil {
		return 0, StatusPending, nil, err
	}

	compilerOutput, err := worker.Compile()
	if err != nil {
		return 0, StatusPending, nil, err
	}

	if compilerOutput.ExitCode != 0 {
		if compilerOutput.CompileLog == nil {
			return 0, StatusSE, &WorkerOutput{Compiler: compilerOutput}, nil
		}
		return 0, StatusCE, &WorkerOutput{Compiler: compilerOutput}, nil
	}

	runnerOutput, err := worker.Run(int(settings.Limits.CPUTime / 1000))
	if err != nil {
		return 0, StatusPending, nil, err
	}

	score := 0
	status := StatusAC
	for i, t := range settings.Tests {
		testCase := runnerOutput.Testcases[i]
		answerCase := answer.Testcases[i]
		if (testCase.SystemOut.Content != answerCase.SystemOut.Content) ||
			(testCase.Failure != nil && testCase.Failure.Message == "Failed") {
			testCase.Status = string(StatusWA)
			if status != StatusTLE && status != StatusRE {
				status = StatusWA
			}
			continue
		}
		if testCase.Time > float64(settings.Limits.TotalTime) ||
			(testCase.Failure != nil && testCase.Failure.Message == "Timeout") {
			testCase.Status = string(StatusTLE)
			if status != StatusRE {
				status = StatusTLE
			}
			continue
		}
		if testCase.Failure == nil {
			testCase.Status = string(StatusAC)
			score += t.Score
			continue
		}
		testCase.Status = string(StatusRE)
		status = StatusRE
	}

	return score, status, &WorkerOutput{Compiler: compilerOutput, TestResults: runnerOutput}, nil
}
