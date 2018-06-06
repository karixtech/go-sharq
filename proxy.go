package sharq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

const (
	UserAgent = "go-sharq"
)

type ProxyClient struct {
	// HTTP client used to communicate with the API.
	client *http.Client

	// Base URL for API requests. This should always be specified without the trailing slash.
	baseURL *url.URL
}

func NewProxyClient(URL string) *ProxyClient {

	client := http.DefaultClient

	url, _ := url.Parse(fmt.Sprintf(URL))

	c := &ProxyClient{client: client, baseURL: url}

	return c

}

type serverResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (c *ProxyClient) BulkEnqueue(e []EnqueueRequest) []EnqueueResponse {
	return bulkEnqueue(c, e)
}

func (c *ProxyClient) Enqueue(e *EnqueueRequest) EnqueueResponse {

	var aResp EnqueueResponse
	var sResp serverResponse
	// Default values
	aResp.JobID = e.JobID
	aResp.Status = "failed"

	queueType := e.QueueType
	queueID := e.QueueID

	if e.Options != nil {
		fmt.Println("Warning: Sharq Proxy does not support requeue_limit parameter")
	}

	enqueueURL, err := url.Parse(fmt.Sprintf(c.baseURL.String() + "/enqueue/" + queueType + "/" + queueID + "/"))
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

	switch resp.StatusCode {
	case http.StatusCreated:
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			aResp.Error = err
			return aResp
		}
		err = json.Unmarshal(bodyBytes, &sResp)
		if err != nil {
			fmt.Println(err)
			aResp.Error = err
			return aResp
		}
		aResp.Status = sResp.Status
	case http.StatusBadRequest:
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println(err)
			aResp.Error = err
			return aResp
		}
		err = json.Unmarshal(bodyBytes, &sResp)
		if err != nil {
			fmt.Println(err)
			aResp.Error = err
			return aResp
		}
		aResp.Error = ProxySharqError{
			Message: sResp.Message,
		}
	default:
		aResp.Error = errors.New("Could not create")
	}

	return aResp
}

func (c *ProxyClient) Dequeue(queueType string) (*DequeueResponse, error) {
	var aResp DequeueResponse

	dequeueURL, err := url.Parse(fmt.Sprintf(
		c.baseURL.String() + "/dequeue/" + queueType + "/"))
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

func (c *ProxyClient) Finish(queueType, queueID, jobID string) error {
	finishURL, err := url.Parse(fmt.Sprintf(
		c.baseURL.String() + "/finish/" + queueType + "/" +
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
