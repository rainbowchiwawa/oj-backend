package sandbox

import (
	"oj/server/sandbox/resources"
)

const (
	COMPILER_IMG_NAME = "oj-compiler:latest"
	RUNNER_IMG_NAME   = "oj-runner:latest"
)

func Init() {
	resources.Init()
}
