package sharq

import (
	"strconv"
	"sync"
	"time"

	"github.com/vmihailenco/msgpack"
)

func isValidInterval(interval int) bool {
	if interval <= 0 {
		return false
	}
	return true
}

func isValidIdentifier(identifier string) bool {
	// Check for length
	if len(identifier) > 100 || len(identifier) < 1 {
		return false
	}
	// Check for content
	for _, c := range identifier {
		if (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c != '_') && (c != '-') {
			return false
		}
	}
	return true
}

func isValidRequeueLimit(requeue_limit int) bool {
	if requeue_limit <= -2 {
		return false
	}
	return true
}

func serializePayload(payload interface{}) ([]byte, error) {
	return msgpack.Marshal(payload)
}

func deserializePayload(payload string) (interface{}, error) {
	var obj interface{}
	err := msgpack.Unmarshal([]byte(payload), &obj)
	return obj, err
}

func generateEpoch() string {
	return strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)
}

// Common bulk enqueue method which can be called from all client implementations
func bulkEnqueue(c Sharq, e []EnqueueRequest) []EnqueueResponse {
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
				JobID:     ber.JobID,
				Interval:  ber.Interval,
				Payload:   ber.Payload,
				QueueType: ber.QueueType,
				QueueID:   ber.QueueID,
			}
			tempR[request_index] = c.Enqueue(er)
		}(i)
	}

	wg.Wait()

	return tempR
}
