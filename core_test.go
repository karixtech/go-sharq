package sharq

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/go-redis/redis"
	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var testCoreClient *CoreClient
var testQueueID string = "johndoe"
var testQueueType string = "sms"
var testPayload1 map[string]interface{} = map[string]interface{}{
	"to":      "1000000000",
	"message": "Hello, world",
}
var testPayload2 map[string]interface{} = map[string]interface{}{
	"to":      "1000000001",
	"message": "Hello, SharQ",
}
var testRequeueLimit5 int = 5
var testRequeueLimitNeg1 int = -1
var testRequeueLimit0 int = 0

func init() {
	config := NewCoreClientConfig()
	config.JobExpireInterval = 5000
	config.JobRequeueInterval = 5000
	config.DefaultJobRequeueLimit = -1
	config.RedisOptions = &redis.UniversalOptions{
		Addrs: []string{"127.0.0.1:6379"},
		DB:    0,
	}
	testCoreClient = NewCoreClient(config)
}

func cleanUp() {
	testCoreClient.redisclient.FlushDB()
}

func newJobID() string {
	return uuid.Must(uuid.NewV4(), nil).String()
}

func TestCoreEnqueue(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	resp := testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	assert.Equal(t, "queued", resp.Status)
}

func TestCoreEnqueueJobQueue(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	queue_name := fmt.Sprintf(
		"%s:%s:%s", testCoreClient.config.KeyPrefix,
		testQueueType, testQueueID)

	job_queue, err := testCoreClient.redisclient.LRange(queue_name, int64(-1), int64(-1)).Result()
	assert.NoError(t, err)
	assert.Equal(t, job_queue, []string{job_id})
}

func TestCoreEnqueuePayload(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	payload_map_name := fmt.Sprintf("%s:payload", testCoreClient.config.KeyPrefix)
	payload_map_key := fmt.Sprintf("%s:%s:%s", testQueueType, testQueueID, job_id)
	raw_payload, err := testCoreClient.redisclient.HGet(payload_map_name, payload_map_key).Result()
	assert.NoError(t, err)
	// decode the payload from msgpack to dictionary
	payload, err := deserializePayload(raw_payload[1 : len(raw_payload)-1])
	assert.Equal(t, testPayload1, payload)
}

func TestCoreEnqueueInterval(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	interval_map_name := fmt.Sprintf("%s:interval", testCoreClient.config.KeyPrefix)
	interval_map_key := fmt.Sprintf("%s:%s", testQueueType, testQueueID)
	interval, err := testCoreClient.redisclient.HGet(interval_map_name, interval_map_key).Result()
	assert.NoError(t, err)
	assert.Equal(t, "10000", interval)
}

func TestCoreEnqueueRequeueLimit(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
		// without a requeue limit parameter
	})
	requeue_limit_map_name := fmt.Sprintf(
		"%s:%s:%s:requeues_remaining", testCoreClient.config.KeyPrefix,
		testQueueType, testQueueID)

	requeues_remaining, err := testCoreClient.redisclient.HGet(
		requeue_limit_map_name, job_id).Result()
	assert.NoError(t, err)
	assert.Equal(t, "-1", requeues_remaining)

	job_id = newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
		Options: &EnqueueRequestOptions{
			RequeueLimit: testRequeueLimit5,
		},
	})
	requeues_remaining, err = testCoreClient.redisclient.HGet(
		requeue_limit_map_name, job_id).Result()
	assert.NoError(t, err)
	assert.Equal(t, "5", requeues_remaining)
}

func TestCoreEnqueueReadySet(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	start_time := generateEpoch()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	end_time := generateEpoch()
	sorted_set_name := fmt.Sprintf("%s:%s", testCoreClient.config.KeyPrefix, testQueueType)
	// Assert sorted set exists
	val, err := testCoreClient.redisclient.Exists(sorted_set_name).Result()
	assert.NoError(t, err)
	assert.Equal(t, val, int64(1))
	// Assert sorted set contents
	queue_id_list, err := testCoreClient.redisclient.ZRangeByScore(
		sorted_set_name, redis.ZRangeBy{
			Min: start_time,
			Max: end_time,
		}).Result()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(queue_id_list))
	assert.Equal(t, testQueueID, queue_id_list[0])
}

func TestCoreEnqueueQueueTypeReadySet(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	queue_type_ready_set, err := testCoreClient.redisclient.SMembers(
		fmt.Sprintf("%s:ready:queue_type", testCoreClient.config.KeyPrefix),
	).Result()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(queue_type_ready_set))
	assert.Equal(t, testQueueType, queue_type_ready_set[0])
}

func TestCoreEnqueueQueueTypeActiveSet(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	queue_type_active_set, err := testCoreClient.redisclient.SMembers(
		fmt.Sprintf("%s:active:queue_type", testCoreClient.config.KeyPrefix),
	).Result()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(queue_type_active_set))
}

func TestCoreEnqueueMetricsGlobalEnqueueCounter(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	timestamp := generateEpoch()
	timestamp_minute := strconv.FormatInt(decimal.RequireFromString(timestamp).Div(decimal.RequireFromString("60000")).Floor().Mul(decimal.RequireFromString("60000")).IntPart(), 10)
	counter_value, err := testCoreClient.redisclient.Get(
		fmt.Sprintf("%s:enqueue_counter:%s", testCoreClient.config.KeyPrefix, timestamp_minute),
	).Result()
	assert.NoError(t, err)
	assert.Equal(t, "1", counter_value)
}

func TestCoreEnqueueMetricsPerQueueEnqueueCounter(t *testing.T) {
	cleanUp()
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	testCoreClient.Dequeue(testQueueType)
	timestamp := generateEpoch()
	timestamp_minute := strconv.FormatInt(decimal.RequireFromString(timestamp).Div(decimal.RequireFromString("60000")).Floor().Mul(decimal.RequireFromString("60000")).IntPart(), 10)
	counter_value, err := testCoreClient.redisclient.Get(
		fmt.Sprintf(
			"%s:%s:%s:enqueue_counter:%s", testCoreClient.config.KeyPrefix,
			testQueueType, testQueueID, timestamp_minute),
	).Result()
	assert.NoError(t, err)
	assert.Equal(t, "1", counter_value)
}

func TestCoreEnqueueSecondJobStatus(t *testing.T) {
	cleanUp()

	// Job 1
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	// Job 2
	job_id = newJobID()
	resp := testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  20000,
		Payload:   testPayload2,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	assert.Equal(t, "queued", resp.Status)
}

func TestCoreEnqueueSecondJobInJobQueue(t *testing.T) {
	cleanUp()

	// Job 1
	job_id_1 := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id_1,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	// Job 2
	job_id_2 := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id_2,
		Interval:  20000,
		Payload:   testPayload2,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	queue_name := fmt.Sprintf("%s:%s:%s", testCoreClient.config.KeyPrefix, testQueueType, testQueueID)
	// Assert queue length
	queue_len, err := testCoreClient.redisclient.LLen(queue_name).Result()
	assert.NoError(t, err)
	assert.Equal(t, queue_len, int64(2))
	// Assert queue contents
	last_elem, err := testCoreClient.redisclient.LRange(queue_name, int64(0), int64(-1)).Result()
	assert.NoError(t, err)
	assert.Equal(t, last_elem, []string{job_id_1, job_id_2})

}

func TestCoreEnqueueSecondJobPayload(t *testing.T) {
	cleanUp()

	// Job 1
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	// Job 2
	job_id = newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  20000,
		Payload:   testPayload2,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	payload_map_name := fmt.Sprintf("%s:payload", testCoreClient.config.KeyPrefix)
	payload_map_key := fmt.Sprintf("%s:%s:%s", testQueueType, testQueueID, job_id)
	raw_payload, err := testCoreClient.redisclient.HGet(payload_map_name, payload_map_key).Result()
	assert.NoError(t, err)
	// decode the payload from msgpack to dictionary
	payload, err := deserializePayload(raw_payload[1 : len(raw_payload)-1])
	assert.Equal(t, testPayload2, payload)
}

func TestCoreEnqueueSecondJobInterval(t *testing.T) {
	cleanUp()

	// Job 1
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	// Job 2
	job_id = newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  20000,
		Payload:   testPayload2,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	interval_map_name := fmt.Sprintf("%s:interval", testCoreClient.config.KeyPrefix)
	interval_map_key := fmt.Sprintf("%s:%s", testQueueType, testQueueID)
	interval, err := testCoreClient.redisclient.HGet(interval_map_name, interval_map_key).Result()
	assert.NoError(t, err)
	// Assert interval has been changed
	assert.Equal(t, "20000", interval)
}

func TestCoreEnqueueSecondJobReadySet(t *testing.T) {
	cleanUp()

	// Job 1
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	// sleeping for 500ms to ensure that the time difference between
	// two enqueues is measurable for the test cases.
	time.Sleep(500 * time.Millisecond)

	// Job 2
	job_id = newJobID()
	start_time := generateEpoch()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  20000,
		Payload:   testPayload2,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	end_time := generateEpoch()

	sorted_set_name := fmt.Sprintf("%s:%s", testCoreClient.config.KeyPrefix, testQueueType)
	// Assert sorted set exists
	val, err := testCoreClient.redisclient.Exists(sorted_set_name).Result()
	assert.NoError(t, err)
	assert.Equal(t, val, int64(1))
	// Assert sorted set contents
	queue_id_list, err := testCoreClient.redisclient.ZRangeByScore(
		sorted_set_name, redis.ZRangeBy{
			Min: start_time,
			Max: end_time,
		}).Result()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(queue_id_list))
}

func TestCoreEnqueueSecondJobQueueTypeReadySet(t *testing.T) {
	cleanUp()

	// Job 1
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	// Job 2
	job_id = newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  20000,
		Payload:   testPayload2,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	queue_type_ready_set, err := testCoreClient.redisclient.SMembers(
		fmt.Sprintf("%s:ready:queue_type", testCoreClient.config.KeyPrefix),
	).Result()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(queue_type_ready_set))
	assert.Equal(t, testQueueType, queue_type_ready_set[0])
}

func TestCoreEnqueueSecondJobQueueTypeActiveSet(t *testing.T) {
	cleanUp()

	// Job 1
	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	// Job 2
	job_id = newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  20000,
		Payload:   testPayload2,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})

	queue_type_active_set, err := testCoreClient.redisclient.SMembers(
		fmt.Sprintf("%s:active:queue_type", testCoreClient.config.KeyPrefix),
	).Result()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(queue_type_active_set))
}

func Test1(t *testing.T) {
	cleanUp()

	job_id := newJobID()
	testCoreClient.Enqueue(&EnqueueRequest{
		JobID:     job_id,
		Interval:  10000,
		Payload:   testPayload1,
		QueueType: testQueueType,
		QueueID:   testQueueID,
	})
	testCoreClient.Dequeue(testQueueType)
	err := testCoreClient.Finish(testQueueType, testQueueID, job_id)
	assert.NoError(t, err)
}
