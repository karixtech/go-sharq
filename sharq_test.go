package sharq

import (
	"testing"

	"github.com/stretchr/testify/assert"
	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

const (
	URL = "https://api.sharq-server.com"
)

func TestEnqueue(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://api.sharq-server.com/enqueue/sms/1/",
		httpmock.NewStringResponder(201, `{"status": "queued"}`))

	client := NewClient(URL)

	er := &EnqueueRequest{JobID: "123-123", Interval: 4, Payload: map[string]string{"hello": "world", "foo": "bar"}}

	enqueueResponse, _ := client.Enqueue(er, "sms", "1")

	assert.Equal(t, enqueueResponse.JobID, er.JobID)
	assert.Equal(t, enqueueResponse.Status, "queued")
}

func TestBulkEnqueueWithSinglePayload(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://api.sharq-server.com/enqueue/sms/1/",
		httpmock.NewStringResponder(201, `{"status": "queued"}`))

	client := NewClient(URL)

	ber := []BulkEnqueueRequest{
		{JobID: "134-145", Interval: 4, Payload: map[string]string{"hello": "world", "foo": "bar"}, QueueType: "sms", QueueID: "1"},
	}

	enqueueResponse, _ := client.BulkEnqueue(ber)

	assert.Equal(t, enqueueResponse[0].JobID, ber[0].JobID)
	assert.Equal(t, enqueueResponse[0].Status, "queued")
}

func TestBulkEnqueue(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://api.sharq-server.com/enqueue/sms/1/",
		httpmock.NewStringResponder(201, `{"status": "queued"}`))

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
