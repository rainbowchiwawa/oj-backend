package sandbox

import "fmt"

const (
	COMPILER_IMG_NAME = "oj-compiler:latest"
	RUNNER_IMG_NAME   = "oj-runner:latest"
	MAX_JOB_COUNT     = 20
	MAX_WORKER_COUNT  = 5
)

var updater func(WorkerOutput)
var jobQueue = make(chan WorkerInput, MAX_JOB_COUNT)
var outputQueue = make(chan WorkerOutput)
var semaphore = make(chan struct{}, MAX_WORKER_COUNT)

func acquire() func() {
	semaphore <- struct{}{}
	return func() { <-semaphore }
}

func PushJob(payload WorkerInput) {
	jobQueue <- payload
}

func PopResult() WorkerOutput {
	return <-outputQueue
}

func consume() {
	release := acquire()
	defer release()
	input := <-jobQueue
	output, err := createWorker(input)
	if err != nil {
		fmt.Println("worker error:", err)
		outputQueue <- WorkerOutput{
			SubmissionId: input.SubmissionId,
			Score:        0,
			Status:       StatusError,
			Output:       &WorkerLogs{},
		}
		return
	}
	outputQueue <- *output
}

func init() {
	go func() {
		for {
			consume()
		}
	}()
}
