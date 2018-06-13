package sharq

type EnqueueRequestOptions struct {
	RequeueLimit int
}

type EnqueueRequest struct {
	JobID     string      `json:"job_id"`
	Interval  int         `json:"interval"`
	Payload   interface{} `json:"payload"`
	QueueID   string      `json:"-"`
	QueueType string      `json:"-"`

	Options *EnqueueRequestOptions `json:"-"`
}

type EnqueueResponse struct {
	Status string `json:"status"`
	JobID  string `json:"job_id"`

	Error error `json:"-"`
}

type DequeueResponse struct {
	Status            string      `json:"status"`
	QueueID           string      `json:"queue_id"`
	JobID             string      `json:"job_id"`
	Payload           interface{} `json:"payload"`
	RequeuesRemaining int         `json:"requeues_remaining"`
}
