package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	DefaultBaseURL = "https://api.onlineornot.com"
	UserAgent      = "terraform-provider-onlineornot/1.0.0"
)

// Client is the OnlineOrNot API client
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// Config holds the configuration for the client
type Config struct {
	APIKey  string
	BaseURL string
}

// NewClient creates a new OnlineOrNot API client
func NewClient(config *Config) *Client {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	return &Client{
		BaseURL: baseURL,
		APIKey:  config.APIKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// APIResponse represents the standard OnlineOrNot API response wrapper
type APIResponse[T any] struct {
	Result   T            `json:"result"`
	Success  bool         `json:"success"`
	Errors   []APIError   `json:"errors"`
	Messages []APIMessage `json:"messages"`
}

// APIListResponse represents a paginated list response
type APIListResponse[T any] struct {
	Result     []T          `json:"result"`
	ResultInfo ResultInfo   `json:"result_info"`
	Success    bool         `json:"success"`
	Errors     []APIError   `json:"errors"`
	Messages   []APIMessage `json:"messages"`
}

// ResultInfo contains pagination information
type ResultInfo struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Count      int `json:"count"`
	TotalCount int `json:"total_count"`
}

// APIError represents an error from the API
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// APIMessage represents a message from the API
type APIMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	url := fmt.Sprintf("%s%s", c.BaseURL, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiResp APIResponse[interface{}]
		if err := json.Unmarshal(respBody, &apiResp); err == nil && len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("API error: %s (code: %d)", apiResp.Errors[0].Message, apiResp.Errors[0].Code)
		}
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Get performs a GET request
func (c *Client) Get(path string) ([]byte, error) {
	return c.doRequest(http.MethodGet, path, nil)
}

// Post performs a POST request
func (c *Client) Post(path string, body interface{}) ([]byte, error) {
	return c.doRequest(http.MethodPost, path, body)
}

// Patch performs a PATCH request
func (c *Client) Patch(path string, body interface{}) ([]byte, error) {
	return c.doRequest(http.MethodPatch, path, body)
}

// Put performs a PUT request
func (c *Client) Put(path string, body interface{}) ([]byte, error) {
	return c.doRequest(http.MethodPut, path, body)
}

// Delete performs a DELETE request
func (c *Client) Delete(path string) ([]byte, error) {
	return c.doRequest(http.MethodDelete, path, nil)
}
