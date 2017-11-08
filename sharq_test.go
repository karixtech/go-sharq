package sharq

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	URL = "http://127.0.0.1:8080/api/v1/namespaces/default/services/sharq-server-sharq-server-chart:8081/proxy"
)

func TestEnqueue(t *testing.T) {
	client := NewClient(URL)

	er := &EnqueueRequest{JobID: "123-123", Interval: 4, Payload: map[string]string{"hello": "world", "foo": "bar"}}

	enqueueResponse, err := client.Enqueue(er, "sms", "1")
	if err != nil {
		t.Errorf("Failed to queue: %v", err)
	} else {
		t.Logf("Enqueued: %v\n", enqueueResponse)
	}
}

func TestBulkEnqueueWithSinglePayload(t *testing.T) {

	client := NewClient(URL)

	ber := []BulkEnqueueRequest{
		{JobID: "134-145", Interval: 4, Payload: map[string]string{"hello": "world", "foo": "bar"}, QueueType: "sms", QueueID: "1"},
	}

	enqueueResponse, _ := client.BulkEnqueue(ber)

	assert.Equal(t, enqueueResponse[0].JobID, ber[0].JobID)
	assert.Equal(t, enqueueResponse[0].Status, "queued")
}

func TestBulkEnqueue(t *testing.T) {

	client := NewClient(URL)

	ber := []BulkEnqueueRequest{
		{JobID: "134-145", Interval: 4, Payload: map[string]string{"hello": "world", "foo": "bar"}, QueueType: "sms", QueueID: "1"},
		{JobID: "136-147", Interval: 4, Payload: map[string]string{"egg": "spam", "foo": "bar"}, QueueType: "sms", QueueID: "1"},
	}

	enqueueResponse, _ := client.BulkEnqueue(ber)

	if enqueueResponse[0].JobID == ber[0].JobID {
		assert.Equal(t, enqueueResponse[0].JobID, ber[0].JobID)
		assert.Equal(t, enqueueResponse[0].Status, "queued")

		assert.Equal(t, enqueueResponse[1].JobID, ber[1].JobID)
		assert.Equal(t, enqueueResponse[1].Status, "queued")
	} else {
		assert.Equal(t, enqueueResponse[0].JobID, ber[1].JobID)
		assert.Equal(t, enqueueResponse[0].Status, "queued")

		assert.Equal(t, enqueueResponse[1].JobID, ber[0].JobID)
		assert.Equal(t, enqueueResponse[1].Status, "queued")
	}
}
