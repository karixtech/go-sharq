package sharq

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync"
)

const (
	UserAgent = "go-sharq"
)

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

type Client struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests. This should always be specified without the trailing slash.
	BaseURL *url.URL
}

func NewClient(URL string) *Client {

	client := http.DefaultClient

	baseURL, _ := url.Parse(fmt.Sprintf(URL))

	c := &Client{client: client, BaseURL: baseURL}

	return c

}

func (c *Client) BulkEnqueue(e []BulkEnqueueRequest) []EnqueueResponse {
	num_requests := len(e)

	var wg sync.WaitGroup
	wg.Add(num_requests)

	// As opposed to previous pattern this concurrency pattern allows
	// request and response arrays to be in same order
	tempR := make([]EnqueueResponse, num_requests)

	for i := 0; i < num_requests; i++ {
		go func(request_index int) {
			defer wg.Done()
			ber := e[request_index]
			er := &EnqueueRequest{
				JobID:    ber.JobID,
				Interval: ber.Interval,
				Payload:  ber.Payload,
			}
			tempR[request_index] = c.Enqueue(er, ber.QueueType, ber.QueueID)
		}(i)
	}

	wg.Wait()

	return tempR
}

func (c *Client) Enqueue(
	e *EnqueueRequest, queueType string, queueID string) EnqueueResponse {

	var aResp EnqueueResponse
	// Default values
	aResp.JobID = e.JobID
	aResp.Status = "failed"

	enqueueURL, err := url.Parse(fmt.Sprintf(c.BaseURL.String() + "/enqueue/" + queueType + "/" + queueID + "/"))
	if err != nil {
		log.Println(err)
		aResp.Error = err
		return aResp
	}

	jsonValue, err := json.Marshal(e)
	if err != nil {
		fmt.Printf("Error: %v", err)
		aResp.Error = err
		return aResp
	}

	req, err := http.NewRequest("POST", enqueueURL.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("Error creating request object: %v", err)
		aResp.Error = err
		return aResp
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", UserAgent)

	resp, err := c.client.Do(req)

	if err != nil {
		fmt.Println(err)
		aResp.Error = err
		return aResp
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			aResp.Error = err
			return aResp
		}

		err = json.Unmarshal(bodyBytes, &aResp)
		if err != nil {
			fmt.Println(err)
			aResp.Error = err
			return aResp
		}
	} else {
		aResp.Error = errors.New("Could not create")
		return aResp
	}

	return aResp
}

// func (c *Client) Dequeue(queueType string) (*DequeueResponse, error) {
// 	req, err := c.NewRequest("GET", "dequeue/"+queueType+"/", nil)
//
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	req.Header.Add("Content-Type", "application/json")
//
// 	aResp := &DequeueResponse{}
//
// 	resp, err := c.Do(req, aResp)
// }
