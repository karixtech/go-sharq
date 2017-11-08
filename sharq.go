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
	JobID   string `json:job_id"`
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

func (c *Client) BulkEnqueue(e []BulkEnqueueRequest) ([]EnqueueResponse, error) {

	queueMessages := make(chan EnqueueResponse)

	var wg sync.WaitGroup

	go monitorWorker(&wg, queueMessages)

	// Add the number of requests to the WG
	wg.Add(len(e))

	for _, eRequest := range e {
		go c.queueToSharq(eRequest, &wg, queueMessages)
	}

	var tempR []EnqueueResponse

	for response := range queueMessages {
		tempR = append(tempR, response)
	}

	return tempR, nil

}

func (c *Client) queueToSharq(e BulkEnqueueRequest, wg *sync.WaitGroup, cs chan EnqueueResponse) {
	defer wg.Done()

	er := &EnqueueRequest{JobID: e.JobID, Interval: e.Interval, Payload: e.Payload}
	enqueueResponse, err := c.Enqueue(er, e.QueueType, e.QueueID)

	if err != nil {
		log.Fatalln(err)
	}

	cs <- enqueueResponse

}

func monitorWorker(wg *sync.WaitGroup, cs chan EnqueueResponse) {
	wg.Wait()
	close(cs)
}

func (c *Client) Enqueue(e *EnqueueRequest, queueType string, queueID string) (EnqueueResponse, error) {
	var aResp EnqueueResponse

	enqueueURL, err := url.Parse(fmt.Sprintf(c.BaseURL.String() + "/enqueue/" + queueType + "/" + queueID + "/"))
	if err != nil {
		log.Fatalln(err)
	}

	jsonValue, err := json.Marshal(e)
	if err != nil {
		fmt.Printf("Error: %v", err)
		return aResp, err
	}

	req, err := http.NewRequest("POST", enqueueURL.String(), bytes.NewBuffer(jsonValue))
	if err != nil {
		fmt.Printf("Error creating request object: %v", err)
		return aResp, err
	}

	req.Header.Add("Content-Type", "application/json")

	resp, err := c.client.Do(req)

	if err != nil {
		fmt.Println(err)
		return aResp, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {

		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			return aResp, err
		}

		err = json.Unmarshal(bodyBytes, &aResp)
		if err != nil {
			fmt.Println(err)
			return aResp, err
		}
		aResp.JobID = e.JobID
	} else {
		return aResp, errors.New("Could not create")
	}

	return aResp, nil
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
