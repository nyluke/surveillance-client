package go2rtc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type StreamSource struct {
	Name    string `json:"name"`
	URL     string `json:"url,omitempty"`
}

// AddStream registers a stream with go2rtc via its REST API.
// go2rtc expects query parameters: PUT /api/streams?name=<name>&src=<rtsp_url>
func (c *Client) AddStream(name, srcURL string) error {
	params := url.Values{}
	params.Set("name", name)
	params.Set("src", srcURL)

	req, err := http.NewRequest(http.MethodPut, c.baseURL+"/api/streams?"+params.Encode(), nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("go2rtc add stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("go2rtc add stream %q: status %d", name, resp.StatusCode)
	}
	return nil
}

// DeleteStream removes a stream from go2rtc
func (c *Client) DeleteStream(name string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/api/streams?name="+name, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("go2rtc delete stream: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

// ListStreams returns all currently registered streams
func (c *Client) ListStreams() (map[string]any, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/api/streams")
	if err != nil {
		return nil, fmt.Errorf("go2rtc list streams: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// Healthy checks if go2rtc is responding
func (c *Client) Healthy() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/api")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
