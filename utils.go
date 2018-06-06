package sharq

import (
	"sync"
)

// Common bulk enqueue method which can be called from all client implementations
func bulkEnqueue(c Sharq, e []BulkEnqueueRequest) []EnqueueResponse {
	num_requests := len(e)

	var wg sync.WaitGroup
	wg.Add(num_requests)

	// As opposed to previous pattern this concurrency pattern allows
	// request and response arrays to be in same order
	tempR := make([]EnqueueResponse, num_requests)

	for i := 0; i < num_requests; i++ {
		go func(request_index int) {
			defer wg.Done()
			ber := e[request_index]
			er := &EnqueueRequest{
				JobID:    ber.JobID,
				Interval: ber.Interval,
				Payload:  ber.Payload,
			}
			tempR[request_index] = c.Enqueue(er, ber.QueueType, ber.QueueID)
		}(i)
	}

	wg.Wait()

	return tempR
}
