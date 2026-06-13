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


func PushJob(payload WorkerInput) error {
	select {
	case jobQueue <- payload:
		return nil
	default:
		return fmt.Errorf("job queue is full")
	}
}

func PopResult() WorkerOutput {
	return <-outputQueue
}

func consume() {
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
	for i := 0; i < MAX_WORKER_COUNT; i++ {
		go func() {
			for {
				consume()
			}
		}()
	}
}
