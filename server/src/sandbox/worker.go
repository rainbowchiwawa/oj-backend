package sandbox

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"oj/server/utility"
	"os"
	"path/filepath"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type CompilerOutput struct {
	ConfigLog  *string
	CompileLog *string
	ExitCode   int64
}

type RunnerOutput struct {
	OutputLog  *string
	UserOutput *string
	ExitCode   int64
}

type WorkerOutput struct {
	Compiler *CompilerOutput
	Runner   *RunnerOutput
}

func GetSubmissionPath(submissionId string) string {
	return filepath.Join(utility.EnvData.BasePath, "submissions", submissionId)
}

func compile(ctx context.Context, moby *client.Client, submissionId string) (CompilerOutput, error) {

	config := container.Config{
		Image:           COMPILER_IMG_NAME,
		NetworkDisabled: true,
		Cmd:             []string{"cmake", "-S", "src", "-B", "build", "-G", "Ninja", ">", "config.log", "2>&1", "&&", "cmake", "--build", "build", "--verbose", ">", "compile.log", "2>&1"},
		WorkingDir:      "/workspace",
	}

	hostConfig := container.HostConfig{
		Binds: []string{fmt.Sprintf("%s/submissions/%s:/workspace", utility.EnvData.BindBasePath, submissionId)},
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

	opt := client.ContainerCreateOptions{
		Name:       "oj-compiler:" + submissionId,
		Config:     &config,
		HostConfig: &hostConfig,
	}

	createResult, err := moby.ContainerCreate(ctx, opt)
	if err != nil {
		return CompilerOutput{}, err
	}

	containerId := createResult.ID
	defer moby.ContainerRemove(ctx, containerId, client.ContainerRemoveOptions{RemoveVolumes: false})

	_, err = moby.ContainerStart(ctx, containerId, client.ContainerStartOptions{})
	if err != nil {
		return CompilerOutput{}, err
	}

	waitResult := moby.ContainerWait(ctx, containerId, client.ContainerWaitOptions{})
	res, err := <-waitResult.Result, <-waitResult.Error
	if err != nil {
		return CompilerOutput{}, err
	}

	basePath := GetSubmissionPath(submissionId)

	configLogBytes, err := os.ReadFile(filepath.Join(basePath, "config.log"))
	if err != nil {
		return CompilerOutput{}, err
	}
	configLog := string(configLogBytes)

	compileLogBytes, err := os.ReadFile(filepath.Join(basePath, "compile.log"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return CompilerOutput{ConfigLog: &configLog, ExitCode: res.StatusCode}, nil
		}
		return CompilerOutput{}, err
	}
	compileLog := string(compileLogBytes)

	return CompilerOutput{ConfigLog: &configLog, CompileLog: &compileLog, ExitCode: res.StatusCode}, nil
}

func run(ctx context.Context, moby *client.Client, submissionId string) (RunnerOutput, error) {

	shellCommands := []string{
		"RUN apk add --no-cache libstdc++",
		"RUN adduser -D runner",
	}

	config := container.Config{
		Image:           COMPILER_IMG_NAME,
		NetworkDisabled: true,
		Cmd:             []string{"$(find . -maxdepth 1 -type f -executable | head -n 1)", ">", "user_output.txt", "2>", "output.log"},
		WorkingDir:      "/workspace",
		Shell:           shellCommands,
		User:            "runner",
	}

	hostConfig := container.HostConfig{
		Binds: []string{fmt.Sprintf("%s/submissions/%s/src/build:/workspace", utility.EnvData.BindBasePath, submissionId)},
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

	opt := client.ContainerCreateOptions{
		Name:       "oj-runner:" + submissionId,
		Config:     &config,
		HostConfig: &hostConfig,
	}

	createResult, err := moby.ContainerCreate(ctx, opt)
	if err != nil {
		return RunnerOutput{}, err
	}

	containerId := createResult.ID
	defer moby.ContainerRemove(ctx, containerId, client.ContainerRemoveOptions{RemoveVolumes: false})

	_, err = moby.ContainerStart(ctx, containerId, client.ContainerStartOptions{})
	if err != nil {
		return RunnerOutput{}, err
	}

	waitResult := moby.ContainerWait(ctx, containerId, client.ContainerWaitOptions{})
	res, err := <-waitResult.Result, <-waitResult.Error
	if err != nil {
		return RunnerOutput{}, err
	}

	basePath := GetSubmissionPath(submissionId)

	outputLogBytes, err := os.ReadFile(filepath.Join(basePath, "output.log"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return RunnerOutput{ExitCode: res.StatusCode}, nil
		}
		return RunnerOutput{}, err
	}
	outputLog := string(outputLogBytes)

	userOutputBytes, err := os.ReadFile(filepath.Join(basePath, "user_output.txt"))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return RunnerOutput{ExitCode: res.StatusCode, OutputLog: &outputLog}, nil
		}
		return RunnerOutput{}, err
	}
	userOutput := string(userOutputBytes)

	return RunnerOutput{ExitCode: res.StatusCode, OutputLog: &outputLog, UserOutput: &userOutput}, nil
}

func CreateWorker(submissionId string) (WorkerOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	moby, err := client.New()
	if err != nil {
		return WorkerOutput{}, err
	}

	compilerOutput, err := compile(ctx, moby, submissionId)
	if err != nil {
		return WorkerOutput{}, err
	}

	if compilerOutput.ExitCode != 0 {
		return WorkerOutput{Compiler: &compilerOutput}, nil
	}

	runnerOutput, err := run(ctx, moby, submissionId)
	if err != nil {
		return WorkerOutput{}, err
	}

	return WorkerOutput{Compiler: &compilerOutput, Runner: &runnerOutput}, nil
}
