package sandbox

import (
	"fmt"
	"oj/server/sandbox/resources"
	"os"
	"sync"

	"github.com/moby/moby/client"
)

const (
	COMPILER_IMG_NAME = "gcc:latest"
	RUNNER_IMG_NAME   = "alpine:latest"
)

var wg sync.WaitGroup

func Init() {
	resources.Init()

	moby, err := client.New()
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer moby.Close()

	wg.Add(1)
	go func() {
		err := prefetch(moby, []string{
			COMPILER_IMG_NAME,
			RUNNER_IMG_NAME,
		})
		if err != nil {
			fmt.Println(err)
			os.Exit(-1)
		}
		wg.Done()
	}()
}
