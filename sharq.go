package sharq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
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

type DequeueResponse struct {
	Status            string      `json:"status"`
	QueueID           string      `json:"queue_id"`
	JobID             string      `json:"job_id"`
	Payload           interface{} `json:"payload"`
	RequeuesRemaining int         `json:"requeues_remaining"`
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

func (c *Client) Dequeue(queueType string) (*DequeueResponse, error) {
	var aResp DequeueResponse
	// TODO: Set default values here

	dequeueURL, err := url.Parse(fmt.Sprintf(
		c.BaseURL.String() + "/dequeue/" + queueType + "/"))
	if err != nil {
		return nil, err
	}

	// Prepare request
	req, err := http.NewRequest("GET", dequeueURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", UserAgent)

	// Perform request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(bodyBytes, &aResp)
		if err != nil {
			return nil, err
		}
	case http.StatusNotFound:
		return nil, nil
	case http.StatusBadRequest:
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read bad request response")
		}
		var errResp struct {
			Message string `json:"message"`
		}
		if err = json.Unmarshal(bodyBytes, &errResp); err == nil {
			return nil, fmt.Errorf("Bad request: %s", errResp.Message)
		} else {
			return nil, errors.Wrap(err, "Bad request")
		}
	default:
		return nil, errors.New("Could not dequeue")
	}

	return &aResp, nil
}

func (c *Client) Finish(queueType, queueID, jobID string) error {
	finishURL, err := url.Parse(fmt.Sprintf(
		c.BaseURL.String() + "/finish/" + queueType + "/" +
			queueID + "/" + jobID + "/"))
	if err != nil {
		return err
	}

	// Prepare request
	req, err := http.NewRequest("POST", finishURL.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", UserAgent)

	// Perform request
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	var aResp struct {
		Status string `json:"status"`
	}

	switch resp.StatusCode {
	case http.StatusOK:
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		err = json.Unmarshal(bodyBytes, &aResp)
		if err != nil {
			return err
		}
	case http.StatusNotFound:
		return errors.New("Job Not Found")
	case http.StatusBadRequest:
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Failed to read bad request response")
		}
		var errResp struct {
			Message string `json:"message"`
		}
		if err = json.Unmarshal(bodyBytes, &errResp); err == nil {
			return fmt.Errorf("Bad request: %s", errResp.Message)
		} else {
			return errors.Wrap(err, "Bad request")
		}
	default:
		return errors.New("Could not dequeue")
	}

	if aResp.Status == "success" {
		return nil
	} else {
		return errors.New("Failure")
	}
}
