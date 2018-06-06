package sharq

import (
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
func NewCoreClientConfig() *CoreClientConfig {
	return &CoreClientConfig{
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

func (c *CoreClient) BulkEnqueue(e []BulkEnqueueRequest) []EnqueueResponse {
	return bulkEnqueue(c, e)
}

func (c *CoreClient) Enqueue(
	e *EnqueueRequest, queueType string, queueID string) EnqueueResponse {

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
	if !isValidIdentifier(queueID) {
		aResp.Error = errors.New("`queue_id` has an invalid value.")
		return aResp
	}
	if !isValidIdentifier(queueType) {
		aResp.Error = errors.New("`queue_type` has an invalid value.")
		return aResp
	}
	// TODO: Allow requeue limit to be set other than default
	requeue_limit := c.config.DefaultJobRequeueLimit
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
		[]string{c.config.KeyPrefix, queueType},
		// Args
		timestamp, queueID, e.JobID, string(serialized_payload),
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

	dequeue_response, err := scripts.DequeueScript.Run(
		c.redisclient,
		// Keys
		[]string{c.config.KeyPrefix, queueType},
		// Args
		timestamp, c.config.JobExpireInterval,
	).Result()
	if err != nil {
		return nil, err
	}

	// TODO: No idea about what to do with dequeue_response
	_ = dequeue_response

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

	finish_response, err := scripts.FinishScript.Run(
		c.redisclient,
		// Keys
		[]string{c.config.KeyPrefix, queueType},
		// Args
		queueID, jobID,
	).Result()
	if err != nil {
		return err
	}

	// TODO: No idea about what to do with finish_response
	_ = finish_response

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
		requeue_response, err := scripts.RequeueScript.Run(
			c.redisclient,
			// Keys
			[]string{c.config.KeyPrefix, queue_type},
			// Args
			timestamp,
		).Result()
		if err != nil {
			last_err = err
		}

		// TODO: No idea about what to do with requeue_response
		// TODO: Finish discarded jobs
		_ = requeue_response
	}

	return last_err
}
