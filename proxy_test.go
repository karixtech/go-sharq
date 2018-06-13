package sharq

import (
	"fmt"
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

	client := NewProxyClient(URL)

	er := &EnqueueRequest{
		JobID:     "123-123",
		Interval:  4,
		Payload:   map[string]string{"hello": "world", "foo": "bar"},
		QueueType: "sms",
		QueueID:   "1",
	}

	enqueueResponse := client.Enqueue(er)

	assert.NoError(t, enqueueResponse.Error)
	assert.Equal(t, enqueueResponse.JobID, er.JobID)
	assert.Equal(t, enqueueResponse.Status, "queued")
}

func TestBulkEnqueueWithSinglePayload(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://api.sharq-server.com/enqueue/sms/1/",
		httpmock.NewStringResponder(201, `{"status": "queued"}`))

	client := NewProxyClient(URL)

	ber := []EnqueueRequest{
		{JobID: "134-145", Interval: 4, Payload: map[string]string{"hello": "world", "foo": "bar"}, QueueType: "sms", QueueID: "1"},
	}

	enqueueResponse := client.BulkEnqueue(ber)

	assert.NoError(t, enqueueResponse[0].Error)
	assert.Equal(t, enqueueResponse[0].JobID, ber[0].JobID)
	assert.Equal(t, enqueueResponse[0].Status, "queued")
}

func TestBulkEnqueue(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST", "https://api.sharq-server.com/enqueue/sms/1/",
		httpmock.NewStringResponder(201, `{"status": "queued"}`))

	client := NewProxyClient(URL)

	ber := []EnqueueRequest{
		{JobID: "134-145", Interval: 4, Payload: map[string]string{"hello": "world", "foo": "bar"}, QueueType: "sms", QueueID: "1"},
		{JobID: "136-147", Interval: 4, Payload: map[string]string{"egg": "spam", "foo": "bar"}, QueueType: "sms", QueueID: "1"},
	}

	enqueueResponse := client.BulkEnqueue(ber)

	if enqueueResponse[0].JobID == ber[0].JobID {
		assert.NoError(t, enqueueResponse[0].Error)
		assert.Equal(t, enqueueResponse[0].JobID, ber[0].JobID)
		assert.Equal(t, enqueueResponse[0].Status, "queued")

		assert.NoError(t, enqueueResponse[1].Error)
		assert.Equal(t, enqueueResponse[1].JobID, ber[1].JobID)
		assert.Equal(t, enqueueResponse[1].Status, "queued")
	} else {
		assert.NoError(t, enqueueResponse[0].Error)
		assert.Equal(t, enqueueResponse[0].JobID, ber[1].JobID)
		assert.Equal(t, enqueueResponse[0].Status, "queued")

		assert.NoError(t, enqueueResponse[1].Error)
		assert.Equal(t, enqueueResponse[1].JobID, ber[0].JobID)
		assert.Equal(t, enqueueResponse[1].Status, "queued")
	}
}

func TestDequeue(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.sharq-server.com/dequeue/sms/",
		httpmock.NewStringResponder(200, `{
				"status": "success",
				"queue_id": "1",
				"job_id": "123-123",
				"payload": {
					"hello": "world",
					"foo": "bar"
				},
				"requeues_remaining": 1
			}`))

	client := NewProxyClient(URL)

	dequeueResponse, err := client.Dequeue("sms")

	assert.NoError(t, err)
	assert.Equal(t, dequeueResponse.JobID, "123-123")
	assert.Equal(t, dequeueResponse.Payload,
		map[string]interface{}{"hello": "world", "foo": "bar"})
	assert.Equal(t, dequeueResponse.QueueID, "1")
	assert.Equal(t, dequeueResponse.RequeuesRemaining, 1)
	assert.Equal(t, dequeueResponse.Status, "success")
}

func TestDequeueNotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://api.sharq-server.com/dequeue/sms/",
		httpmock.NewStringResponder(404, `{
				"status": "failure"
			}`))

	client := NewProxyClient(URL)

	dequeueResponse, err := client.Dequeue("sms")

	// No Jobs in sharq should not raise an error
	assert.Nil(t, dequeueResponse)
	assert.Nil(t, err)
}

func TestFinish(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST",
		"https://api.sharq-server.com/finish/sms/1/123-123/",
		httpmock.NewStringResponder(200,
			`{
				"status": "success"
			}`,
		),
	)

	client := NewProxyClient(URL)

	err := client.Finish("sms", "1", "123-123")

	assert.NoError(t, err)
}

func TestFinishFailure(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST",
		"https://api.sharq-server.com/finish/sms/1/123-123/",
		httpmock.NewStringResponder(200,
			`{
				"status": "failure"
			}`,
		),
	)

	client := NewProxyClient(URL)

	err := client.Finish("sms", "1", "123-123")

	if assert.Error(t, err) {
		assert.Equal(t, "Failure", fmt.Sprintf("%v", err))
	}
}

func TestFinishNotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("POST",
		"https://api.sharq-server.com/finish/sms/1/123-123/",
		httpmock.NewStringResponder(404,
			`{
				"status": "failure"
			}`,
		),
	)

	client := NewProxyClient(URL)

	err := client.Finish("sms", "1", "123-123")

	if assert.Error(t, err) {
		assert.Equal(t, "Job Not Found", fmt.Sprintf("%v", err))
	}
}
