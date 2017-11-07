package sharq

import "testing"

const (
	URL = "http://127.0.0.1:8080/api/v1/namespaces/default/services/sharq-server-sharq-server-chart:8081/proxy"
)

func TestEnqueue(t *testing.T) {
	client := NewClient(URL)

	er := &EnqueueRequest{JobID: "123-123", Interval: 4, Payload: map[string]string{"hello": "world"}}

	enqueueResponse, err := client.Enqueue(er, "sms", "1")
	if err != nil {
		t.Errorf("Failed to queue: %v", err)
	} else {
		t.Logf("Enqueued: %v\n", enqueueResponse)
	}
}
