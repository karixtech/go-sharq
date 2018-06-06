package sharq

//go:generate go-bindata -o scripts/bindata.go -pkg scripts ./lua/...

type Sharq interface {
	Enqueue(e *EnqueueRequest, queueType string, queueID string) EnqueueResponse
	BulkEnqueue(e []BulkEnqueueRequest) []EnqueueResponse
	Dequeue(queueType string) (*DequeueResponse, error)
	Finish(queueType, queueID, jobID string) error
}
