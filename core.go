package sharq

import (
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis"
	"github.com/pkg/errors"

	"github.com/karixtech/go-sharq/scripts"
)

type CoreClientConfig struct {
	JobExpireInterval      int
	JobRequeueInterval     int
	DefaultJobRequeueLimit int

	RedisOptions *redis.UniversalOptions

	KeyPrefix string

	RunRequeue bool
}

// Returns a config object with default values set
func NewCoreClientConfig() CoreClientConfig {
	return CoreClientConfig{
		JobExpireInterval:      1000,
		JobRequeueInterval:     1000,
		DefaultJobRequeueLimit: -1,
		KeyPrefix:              "sharq_server",
	}
}

type CoreClient struct {
	// HTTP client used to communicate with the API.
	redisclient redis.UniversalClient

	config CoreClientConfig
}

func NewCoreClient(config CoreClientConfig) *CoreClient {
	client := redis.NewUniversalClient(config.RedisOptions)

	c := &CoreClient{
		redisclient: client,
		config:      config,
	}

	if c.config.RunRequeue {
		go func() {
			for true {
				c.requeue()
				time.Sleep(time.Duration(c.config.JobRequeueInterval) * time.Millisecond)
			}
		}()
	}

	return c
}

func (c *CoreClient) BulkEnqueue(e []EnqueueRequest) []EnqueueResponse {
	return bulkEnqueue(c, e)
}

func (c *CoreClient) Enqueue(e *EnqueueRequest) EnqueueResponse {

	var aResp EnqueueResponse
	// Default values
	aResp.JobID = e.JobID
	aResp.Status = "failed"

	if !isValidInterval(e.Interval) {
		aResp.Error = errors.New("`interval` has an invalid value.")
		return aResp
	}
	if !isValidIdentifier(e.JobID) {
		aResp.Error = errors.New("`job_id` has an invalid value.")
		return aResp
	}
	if !isValidIdentifier(e.QueueID) {
		aResp.Error = errors.New("`queue_id` has an invalid value.")
		return aResp
	}
	if !isValidIdentifier(e.QueueType) {
		aResp.Error = errors.New("`queue_type` has an invalid value.")
		return aResp
	}
	requeue_limit := c.config.DefaultJobRequeueLimit
	if e.Options != nil {
		requeue_limit = e.Options.RequeueLimit
	}
	if !isValidRequeueLimit(requeue_limit) {
		aResp.Error = errors.New("`requeue_limit` has an invalid value.")
		return aResp
	}

	serialized_payload, err := serializePayload(e.Payload)
	if err != nil {
		aResp.Error = err
		return aResp
	}

	timestamp := generateEpoch()

	_, err = scripts.EnqueueScript.Run(
		c.redisclient,
		// Keys
		[]string{c.config.KeyPrefix, e.QueueType},
		// Args
		timestamp, e.QueueID, e.JobID, string(serialized_payload),
		e.Interval, requeue_limit,
	).Result()
	if err != nil {
		aResp.Error = err
		return aResp
	}
	aResp.Status = "queued"
	return aResp
}

func (c *CoreClient) Dequeue(queueType string) (*DequeueResponse, error) {
	var aResp DequeueResponse

	if !isValidIdentifier(queueType) {
		return nil, errors.New("`queue_type` has an invalid value.")
	}

	timestamp := generateEpoch()

	raw_response, err := scripts.DequeueScript.Run(
		c.redisclient,
		// Keys
		[]string{c.config.KeyPrefix, queueType},
		// Args
		timestamp, c.config.JobExpireInterval,
	).Result()
	if err != nil {
		return nil, err
	}

	dequeue_response, ok := raw_response.([]interface{})
	if !ok {
		return nil, errors.New("Dequeue lua script responded with incorrect type")
	}
	if len(dequeue_response) < 4 {
		// Job not found should return no error and no job
		return nil, nil
	}

	deserialized_payload, err := deserializePayload(dequeue_response[2].(string))
	if err != nil {
		return nil, err
	}

	requeues_remaining, err := strconv.Atoi(dequeue_response[3].(string))
	if err != nil {
		requeues_remaining = 0
	}

	aResp.Status = "success"
	aResp.QueueID = dequeue_response[0].(string)
	aResp.JobID = dequeue_response[1].(string)
	aResp.Payload = deserialized_payload
	aResp.RequeuesRemaining = requeues_remaining

	return &aResp, nil
}

func (c *CoreClient) Finish(queueType, queueID, jobID string) error {
	if !isValidIdentifier(jobID) {
		return errors.New("`job_id` has an invalid value.")
	}
	if !isValidIdentifier(queueID) {
		return errors.New("`queue_id` has an invalid value.")
	}
	if !isValidIdentifier(queueType) {
		return errors.New("`queue_type` has an invalid value.")
	}

	raw_response, err := scripts.FinishScript.Run(
		c.redisclient,
		// Keys
		[]string{c.config.KeyPrefix, queueType},
		// Args
		queueID, jobID,
	).Result()
	if err != nil {
		return err
	}

	finish_response, ok := raw_response.(string)
	if !ok {
		return errors.New("Finish lua script responded with incorrect type")
	}
	if finish_response != "1" {
		return errors.New("Failure")
	}

	return nil
}

func (c *CoreClient) requeue() error {
	timestamp := generateEpoch()

	// active_queue_type_list = self._r.smembers('%s:active:queue_type' % self._key_prefix)
	queue_types, err := c.redisclient.SMembers(
		c.config.KeyPrefix + ":active:queue_type",
	).Result()
	if err != nil {
		return err
	}

	var last_err error = nil
	for _, queue_type := range queue_types {
		raw_response, err := scripts.RequeueScript.Run(
			c.redisclient,
			// Keys
			[]string{c.config.KeyPrefix, queue_type},
			// Args
			timestamp,
		).Result()
		if err != nil {
			last_err = err
			continue
		}
		job_discard_list, ok := raw_response.([]interface{})
		if !ok {
			last_err = errors.New("Requeue lua script responded with incorrect type")
			continue
		}
		for _, job := range job_discard_list {
			job_s := strings.SplitN(job.(string), ":", 2)
			queue_id := job_s[0]
			job_id := job_s[1]
			err = c.Finish(queue_type, queue_id, job_id)
			if err != nil {
				last_err = err
			}
		}
	}

	return last_err
}
