go-sharq
=======

SharQ is a flexible rate limited queueing system built using [Redis](http://redis.io).

go-sharq provides a core client library for both producer and consumer. This is a golang port of python library [sharq](https://github.com/plivo/sharq).
It also provides proxy client to [sharq-server](https://github.com/plivo/sharq-server).

## Installation
Install using go get
```
go get -u github.com/karixtech/go-sharq
```

## Usage

### Initialization
Import the library
```go
import sharq "github.com/karixtech/go-sharq"
```

Initialize the config
```go
// NewCoreClientConfig returns default config
sharq_config := sharq.NewCoreClientConfig()
// Set redis options for universal redis client (supports sentinel and cluster clients)
sharq_config.RedisOptions = &redis.UniversalOptions{
	Addrs:    []string{"127.0.0.1:6379"},
	DB:       0,
}
// Requeue goroutine is disabled by default. Enable it to automatically requeue tasks not completed.
sharq_config.RunRequeue = true
sharq_client = sharq.NewCoreClient(sharq_config)
```

### Enqueue

Enqueues a job into the queue. Every enqueue request is accompanied with an `interval`. The interval specifies the rate limiting capability of SharQ. An interval of 1000ms implies that SharQ will ensure two successful dequeue requests will be separated by 1000ms (interval is the inverse of rate. 1000ms interval means 1 job per second)

```go
response := sharq_client.Enqueue(&sharq.EnqueueRequest{
  JobID:     "cea84623-be35-4368-90fa-7736570dabc4",
  Payload:   map[string]string{"message": "hello, world"},
  Interval:  1000, // in milliseconds
  QueueID:   "user001",
  QueueType: "sms",
})
fmt.Println(response.Status)
fmt.Println(response.JobID)
fmt.Println(response.Error)
```

### Dequeue

Dequeues a job (non-blocking). It returns a job only if available or if it is ready for dequeue (based on the interval set while enqueueing).

```go
response, err := sharq_client.Dequeue("sms")
if err != nil {
  fmt.Println("Unexpected error on sharq dequeue")
  fmt.Println(err)
}
if response == nil {
  fmt.Println("No jobs found in sharq")
}
fmt.Println(response.Status)
fmt.Println(response.QueueID)
fmt.Println(response.JobID)
fmt.Println(response.Payload)
fmt.Println(response.RequeuesRemaining)
```

### Finish

Marks any dequeued job as _succesfully completed_. Any job which does get marked as finished upon dequeue will be re-enqueued into its respective queue after an expiry time (the `JobRequeueInterval` in the config).

```go
err = sharq_client.Finish("sms", "user001", "bb59a2be-3b48-4645-8134-d9181742e3cf")
fmt.Println(err)
```
