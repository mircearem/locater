package db

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Maybe this should be a new function in the storer package
type Client struct {
	remoteAddr string
	client     *http.Client
}

func NewClient(remoteAddr string) *Client {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	return &Client{
		remoteAddr: remoteAddr,
		client:     client,
	}
}

// Get values from the store using keys
func (c *Client) Get(collection string, keys ...string) ([]map[string]string, error) {
	uri := fmt.Sprintf("%s/collection", c.remoteAddr)
	results := make([]map[string]string, 0)

	for _, k := range keys {
		body := bytes.NewBuffer([]byte(k))
		req, err := http.NewRequest("GET", uri, body)
		if err != nil {
			return nil, err
		}
		// Execute request
		resp, err := c.client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		result := make(map[string]string)
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

// Insert key value pairs into the store
func (c *Client) Post(collection string, data ...[]byte) error {
	// Check if data is valid json format
	jmap := make(map[string]string)
	for _, arr := range data {
		err := json.Unmarshal(arr, &jmap)
		if err != nil {
			return err
		}
	}
	// Data is valit json format: {"key":"value"}
	// post it to the collection
	uri := fmt.Sprintf("%s/collection", c.remoteAddr)
	for _, arr := range data {
		body := bytes.NewBuffer(arr)
		req, err := http.NewRequest("POST", uri, body)
		if err != nil {
			return err
		}
		// Execute request
		if _, err = c.client.Do(req); err != nil {
			return err
		}
	}
	return nil
}
