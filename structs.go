package sharq

type EnqueueRequest struct {
	JobID    string      `json:"job_id"`
	Interval int         `json:"interval"`
	Payload  interface{} `json:"payload"`
}

type BulkEnqueueRequest struct {
	JobID     string      `json:"job_id"`
	Interval  int         `json:"interval"`
	Payload   interface{} `json:"payload"`
	QueueID   string      `json:"queue_id"`
	QueueType string      `json:"queue_type"`
}

type EnqueueResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	JobID   string `json:"job_id"`

	// error while sending job to sharq
	Error error `json:"-"`
}

type DequeueResponse struct {
	Status            string      `json:"status"`
	QueueID           string      `json:"queue_id"`
	JobID             string      `json:"job_id"`
	Payload           interface{} `json:"payload"`
	RequeuesRemaining int         `json:"requeues_remaining"`
}
