package sandbox

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"oj/server/parser"
	"oj/server/sandbox/resources"
	"oj/server/utility"
	"oj/server/utility/archiver"
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

func (wo WorkerOutput) Value() (driver.Value, error) {
	return json.Marshal(wo)
}

func (wo *WorkerOutput) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Failed to unmarshal JSONB")
	}
	return json.Unmarshal(bytes, wo)
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

	config := container.Config{
		Image:           COMPILER_IMG_NAME,
		NetworkDisabled: true,
		Cmd: []string{
			"sh", "-c",
			"cmake -S src -B build -G Ninja > config.log 2>&1 && " +
				"cmake --build build --verbose > compile.log 2>&1",
		},
		Labels: map[string]string{"com.docker.compose.progject": "oj-compiler"},
	}

	hostConfig := container.HostConfig{
		Resources: container.Resources{
			CPUCount: 4,
			Memory:   2048 << 20,
		},
		NetworkMode: "none",
	}

	options := client.ContainerCreateOptions{
		Name:       "compiler-" + w.Id,
		Config:     &config,
		HostConfig: &hostConfig,
	}

	basePath := w.Manager.GetBasePath()
	res, err := w.setupContainerAndRun(ContainerSetupOptions{
		CreateOptions: options,
		HostDir:       basePath,
		FilesToImport: []string{
			"spec/",
			"src/",
		},
		FilesToExtract: []string{
			"build",
			"config.log",
			"compile.log",
		},
	})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

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

	config := container.Config{
		Image:           COMPILER_IMG_NAME,
		NetworkDisabled: true,
		Cmd: []string{
			"sh", "-c", fmt.Sprintf("cd build && ctest --timeout %d --output-junit result.xml", timeout),
		},
		Labels: map[string]string{"com.docker.compose.progject": "oj-runner"},
	}

	hostConfig := container.HostConfig{
		Resources: container.Resources{
			CPUCount: 2,
			Memory:   512 << 20,
		},
		NetworkMode: "none",
	}

	options := client.ContainerCreateOptions{
		Name:       "runner-" + w.Id,
		Config:     &config,
		HostConfig: &hostConfig,
	}

	basePath := w.Manager.GetBasePath()
	_, err := w.setupContainerAndRun(ContainerSetupOptions{
		CreateOptions:  options,
		HostDir:        basePath,
		FilesToImport:  []string{"build/"},
		FilesToExtract: []string{"build/result.xml"},
	})
	if err != nil {
		return nil, err
	}

	resultBytes, err := os.ReadFile(filepath.Join(basePath, "build/result.xml"))
	if err != nil {
		return nil, err
	}
	return parser.ParseTestResults(resultBytes)
}

type ContainerSetupOptions struct {
	CreateOptions  client.ContainerCreateOptions
	HostDir        string
	FilesToImport  []string
	FilesToExtract []string
}

func (w Worker) setupContainerAndRun(options ContainerSetupOptions) (*container.WaitResponse, error) {
	createResult, err := w.Moby.ContainerCreate(w.Context, options.CreateOptions)
	if err != nil {
		return nil, err
	}

	containerId := createResult.ID
	defer w.Moby.ContainerRemove(w.Context, containerId, client.ContainerRemoveOptions{RemoveVolumes: false})

	buf, err := makeTarStream(options.HostDir, options.FilesToImport)
	if err != nil {
		return nil, err
	}

	_, err = w.Moby.CopyToContainer(w.Context, containerId, client.CopyToContainerOptions{
		DestinationPath: "/workspace",
		Content:         buf,
	})
	if err != nil {
		return nil, err
	}

	_, err = w.Moby.ContainerStart(w.Context, containerId, client.ContainerStartOptions{})
	if err != nil {
		return nil, err
	}

	waitResult := w.Moby.ContainerWait(w.Context, containerId, client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})

	select {
	case res := <-waitResult.Result:
		logResult, err := w.Moby.ContainerLogs(w.Context, containerId, client.ContainerLogsOptions{
			ShowStdout: true,
		})
		if err != nil {
			return nil, err
		}
		stdbuf := make([]byte, 1024)
		logResult.Read(stdbuf)
		fmt.Println(string(stdbuf))
		for _, f := range options.FilesToExtract {
			copyResult, err := w.Moby.CopyFromContainer(w.Context, containerId, client.CopyFromContainerOptions{
				SourcePath: filepath.Join("/workspace", f),
			})
			if err != nil {
				return nil, err
			}

			fmt.Printf("extracting from %s->%s\n", filepath.Join("/workspace", f), filepath.Join(options.HostDir, filepath.Dir(f)))
			tr := archiver.NewTarReader(copyResult.Content)
			if err = archiver.ExtractTo(tr, filepath.Join(options.HostDir, filepath.Dir(f))); err != nil {
				return nil, err
			}
		}
		return &res, nil
	case err := <-waitResult.Error:
		return nil, err
	}
}

func makeTarStream(srcDir string, files []string) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	tw := archiver.NewTarWriter(&buf)
	defer tw.Close()

	err := archiver.CompressFiles(tw, srcDir, files)
	if err != nil {
		return nil, err
	}

	return &buf, nil
}

func CreateWorker(
	submissionId,
	problemId string,
	settings *parser.ProblemSettings,
	answer *parser.TestResults,
) (int, TestStatus, *WorkerOutput, error) {
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

	fmt.Println("copying test files")
	if err = submissionManager.CopyTestFiles(resources.ProblemManager{Id: problemId}); err != nil {
		return 0, StatusPending, nil, err
	}

	fmt.Println("compiling")
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

	fmt.Println("running")
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
