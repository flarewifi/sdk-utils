package jobque

// Define a Job type
type Job func() (interface{}, error)

// JobQue is the structure that holds the job queue
type JobQue struct {
	jobChannel chan Job
	results    chan JobResult
	stopChan   chan struct{}
}

// JobResult holds the result of a job execution
type JobResult struct {
	Result interface{}
	Err    error
}

// NewJobQue creates and initializes a new JobQue
func NewJobQue() *JobQue {
	jq := &JobQue{
		jobChannel: make(chan Job),
		results:    make(chan JobResult),
		stopChan:   make(chan struct{}),
	}
	go jq.worker()
	return jq
}

// worker processes jobs from the jobChannel
func (jq *JobQue) worker() {
	for {
		select {
		case job := <-jq.jobChannel:
			result, err := job()
			jq.results <- JobResult{Result: result, Err: err}
		case <-jq.stopChan:
			close(jq.results)
			return
		}
	}
}

// Exec adds a job to the queue and waits for the result
func (jq *JobQue) Exec(job Job) (interface{}, error) {
	// Send the job to the worker
	jq.jobChannel <- job

	// Wait for the result
	result := <-jq.results
	return result.Result, result.Err
}
