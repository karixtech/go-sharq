package sharq

type Sharq interface {
	Enqueue(e *EnqueueRequest, queueType string, queueID string) EnqueueResponse
	BulkEnqueue(e []BulkEnqueueRequest) []EnqueueResponse
	Dequeue(queueType string) (*DequeueResponse, error)
	Finish(queueType, queueID, jobID string) error
}
