package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents an API client
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// New creates a new API client
func New(baseURL string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

// SetBaseURL sets the base URL for the API
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// Do performs a request to the API with the given API key and returns the response
func (c *Client) Do(method, path string, body io.Reader, apiKey string) (*http.Response, error) {
	url := c.baseURL + path

	// Add API key to the URL
	if apiKey != "" {
		url += "?key=" + apiKey
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set default content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}

	return resp, nil
}

// CheckAPIKey checks if an API key is valid by making a simple request
func (c *Client) CheckAPIKey(apiKey string) (bool, int, error) {
	// Make a request to the API's key info endpoint
	resp, err := c.Do("GET", "/api-info", nil, apiKey)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	// If the API responds with an error status, the key is invalid or expired
	if resp.StatusCode >= 400 {
		return false, 0, nil
	}

	// Parse the response to get the remaining quota
	// the remaining quote is in field `query_credits`
	// example response:
	// 	{
	//     "scan_credits": 100000,
	//     "usage_limits": {
	//         "scan_credits": -1,
	//         "query_credits": -1,
	//         "monitored_ips": -1
	//     },
	//     "plan": "stream-100",
	//     "https": false,
	//     "unlocked": true,
	//     "query_credits": 100000,
	//     "monitored_ips": 19,
	//     "unlocked_left": 100000,
	//     "telnet": false
	// }
	var keyInfo struct {
		ScanCredits int `json:"scan_credits"`
		UsageLimits struct {
			ScanCredits  int `json:"scan_credits"`
			QueryCredits int `json:"query_credits"`
			MonitoredIPs int `json:"monitored_ips"`
		} `json:"usage_limits"`
		Plan         string `json:"plan"`
		HTTPS        bool   `json:"https"`
		Unlocked     bool   `json:"unlocked"`
		QueryCredits int    `json:"query_credits"`
		MonitoredIPs int    `json:"monitored_ips"`
		UnlockedLeft int    `json:"unlocked_left"`
		Telnet       bool   `json:"telnet"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&keyInfo); err != nil {
		return false, 0, fmt.Errorf("failed to parse response: %w", err)
	}
	if keyInfo.QueryCredits <= 0 {
		return false, 0, nil
	}
	return true, keyInfo.QueryCredits, nil
}
