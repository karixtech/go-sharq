package sharq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	UserAgent = "go-sharq"
)

type EnqueueRequest struct {
	JobID    string      `json:"job_id"`
	Interval int         `json:"interval"`
	Payload  interface{} `json:"payload"`
}

type EnqueueResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Client struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests. This should always be specified with the trailing slash.
	BaseURL *url.URL
}

func NewClient(URL string) *Client {

	client := http.DefaultClient

	baseURL, _ := url.Parse(fmt.Sprintf(URL))

	c := &Client{client: client, BaseURL: baseURL}

	return c

}

func (c *Client) Enqueue(e *EnqueueRequest, queueType string, queueID string) (*EnqueueResponse, error) {
	enqueueURL, err := url.Parse(fmt.Sprintf(c.BaseURL.String() + "/enqueue/" + queueType + "/" + queueID + "/"))

	if err != nil {
		fmt.Printf("%v", err)
	}

	jsonValue, err := json.Marshal(e)
	if err != nil {
		fmt.Printf("Error: %v", err)
	}

	fmt.Println(string(jsonValue))

	if err != nil {
		fmt.Printf("%v", err)
		return nil, err
	}

	req, err := http.NewRequest("POST", enqueueURL.String(), bytes.NewBuffer(jsonValue))

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	aResp := &EnqueueResponse{}

	resp, err := c.client.Do(req)

	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(aResp)

	return aResp, err

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
