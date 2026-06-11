package sandbox

import (
	"bytes"
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"oj/server/parser"
	"oj/server/sandbox/resources"
	"oj/server/utility/archiver"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type WorkerOutput struct {
	SubmissionId string
	Score        int
	Status       TestStatus
	Output       *WorkerLogs
}

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
	ConfigLog  *string `json:"config_log"`
	CompileLog *string `json:"compile_log"`
	ExitCode   int64   `json:"exit_code"`
}

type WorkerInput struct {
	ProblemId    string
	SubmissionId string
	Settings     *parser.ProblemSettings
	Answer       *parser.TestResults
}

type WorkerLogs struct {
	Compiler    *CompilerOutput     `json:"compiler"`
	TestResults *parser.TestResults `json:"test_results"`
}

func (w WorkerLogs) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func (w *WorkerLogs) Scan(value any) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Failed to unmarshal JSONB")
	}
	return json.Unmarshal(bytes, w)
}

type Worker struct {
	Id      string
	Moby    *client.Client
	Context context.Context
	Manager resources.SubmissionManager
}

func (w Worker) compile() (*CompilerOutput, error) {

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
			CPUCount: 2,
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
			"build/",
		},
		FilesToRead: []string{
			"config.log",
			"compile.log",
		},
	})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	configLogBytes, exists := res.FileBytes["config.log"]
	if !exists {
		return nil, fmt.Errorf("config.log not exists")
	}
	configLog := string(configLogBytes)

	compileLogBytes, exists := res.FileBytes["compile.log"]
	if !exists {
		return &CompilerOutput{ConfigLog: &configLog, ExitCode: res.ExitCode}, nil
	}
	compileLog := string(compileLogBytes)

	return &CompilerOutput{ConfigLog: &configLog, CompileLog: &compileLog, ExitCode: res.ExitCode}, nil
}

func (w Worker) run(timeout int) (*parser.TestResults, error) {

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
			CPUCount: 1,
			Memory:   128 << 20,
		},
		NetworkMode: "none",
	}

	options := client.ContainerCreateOptions{
		Name:       "runner-" + w.Id,
		Config:     &config,
		HostConfig: &hostConfig,
	}

	basePath := w.Manager.GetBasePath()
	res, err := w.setupContainerAndRun(ContainerSetupOptions{
		CreateOptions: options,
		HostDir:       basePath,
		FilesToImport: []string{"build/"},
		FilesToRead:   []string{"build/result.xml"},
	})
	if err != nil {
		return nil, err
	}

	resultBytes, exists := res.FileBytes["build/result.xml"]
	if !exists {
		return nil, fmt.Errorf("result.xml not exists")
	}
	return parser.ParseTestResults(resultBytes)
}

type ContainerSetupOptions struct {
	CreateOptions  client.ContainerCreateOptions
	HostDir        string
	FilesToImport  []string
	FilesToExtract []string
	FilesToRead    []string
}

type ContainerRunResult struct {
	ExitCode  int64
	Stdout    []byte
	FileBytes map[string][]byte
}

func (w Worker) setupContainerAndRun(options ContainerSetupOptions) (*ContainerRunResult, error) {
	createResult, err := w.Moby.ContainerCreate(w.Context, options.CreateOptions)
	if err != nil {
		return nil, err
	}

	containerId := createResult.ID
	defer w.Moby.ContainerRemove(w.Context, containerId, client.ContainerRemoveOptions{RemoveVolumes: false})

	buf, err := func() (*bytes.Buffer, error) {
		var buf bytes.Buffer
		tw := archiver.NewTarWriter(&buf)
		defer tw.Close()

		err := archiver.CompressDir(tw, options.HostDir, func(entry archiver.CompressEntry) error {
			for _, f := range options.FilesToImport {
				if entry.Test(f) {
					return entry.Compress()
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		return &buf, nil
	}()
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
		defer logResult.Close()

		var stdbuf bytes.Buffer
		io.Copy(&stdbuf, logResult)
		stdoutBytes := stdbuf.Bytes()
		fmt.Println(string(stdoutBytes))

		copyResult, err := w.Moby.CopyFromContainer(w.Context, containerId, client.CopyFromContainerOptions{
			SourcePath: "/workspace",
		})
		defer copyResult.Content.Close()

		fileBytes := make(map[string][]byte)
		tr := archiver.NewTarReader(copyResult.Content)
		err = archiver.ExtractTo(tr, options.HostDir, "workspace", func(entry archiver.ExtractEntry) error {
			for _, f := range options.FilesToExtract {
				if entry.Test(f) {
					if err := entry.Extract(); err != nil {
						return err
					}
				}
			}
			for _, f := range options.FilesToRead {
				if f == entry.Name {
					bytes, err := entry.Read()
					if err != nil {
						return err
					}
					fileBytes[f] = bytes
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}

		return &ContainerRunResult{
			ExitCode:  res.StatusCode,
			Stdout:    stdoutBytes,
			FileBytes: fileBytes,
		}, nil
	case err := <-waitResult.Error:
		return nil, err
	}
}

func createWorker(payload WorkerInput) (*WorkerOutput, error) {
	submissionManager := resources.SubmissionManager{Id: payload.SubmissionId}
	if err := submissionManager.ExtractZip(); err != nil {
		return nil, err
	}
	defer submissionManager.ClearFiles()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	moby, err := client.New()
	if err != nil {
		return nil, err
	}
	worker := Worker{
		Id:      payload.SubmissionId,
		Moby:    moby,
		Context: ctx,
		Manager: submissionManager,
	}

	fmt.Println("copying test files")
	if err = submissionManager.CopyTestFiles(resources.ProblemManager{Id: payload.ProblemId}); err != nil {
		return nil, err
	}

	fmt.Println("compiling")
	compilerOutput, err := worker.compile()
	if err != nil {
		return nil, err
	}

	if compilerOutput.ExitCode != 0 {
		if compilerOutput.CompileLog == nil {
			return &WorkerOutput{payload.SubmissionId, 0, StatusSE, &WorkerLogs{Compiler: compilerOutput}}, nil
		}
		return &WorkerOutput{payload.SubmissionId, 0, StatusCE, &WorkerLogs{Compiler: compilerOutput}}, nil
	}

	fmt.Println("running")
	runnerOutput, err := worker.run(int(payload.Settings.Limits.CPUTime / 1000))
	if err != nil {
		return nil, err
	}

	score := 0
	status := StatusAC
	for i, t := range payload.Settings.Tests {
		testCase := runnerOutput.Testcases[i]
		answerCase := payload.Answer.Testcases[i]
		if (testCase.SystemOut.Content != answerCase.SystemOut.Content) ||
			(testCase.Failure != nil && testCase.Failure.Message == "Failed") {
			testCase.Status = string(StatusWA)
			if status != StatusTLE && status != StatusRE {
				status = StatusWA
			}
			continue
		}
		if testCase.Time > float64(payload.Settings.Limits.TotalTime) ||
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

	return &WorkerOutput{payload.SubmissionId, score, status, &WorkerLogs{Compiler: compilerOutput, TestResults: runnerOutput}}, nil
}
