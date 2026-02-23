package client

import (
	"encoding/json"
	"fmt"
)

// Webhook represents a webhook
type Webhook struct {
	ID            string   `json:"id,omitempty"`
	URL           string   `json:"url"`
	Description   string   `json:"description,omitempty"`
	Events        []string `json:"events"`
	CheckIDs      []string `json:"check_ids,omitempty"`
	HeartbeatIDs  []string `json:"heartbeat_ids,omitempty"`
	StatusPageIDs []string `json:"status_page_ids,omitempty"`
}

// CreateWebhook creates a new webhook
func (c *Client) CreateWebhook(wh *Webhook) (*Webhook, error) {
	respBody, err := c.Post("/v1/webhooks", wh)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Webhook]
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("API error: %s", apiResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("API request failed")
	}

	return &apiResp.Result, nil
}

// GetWebhook retrieves a webhook by ID
func (c *Client) GetWebhook(id string) (*Webhook, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/webhooks/%s", id))
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Webhook]
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("API error: %s", apiResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("API request failed")
	}

	return &apiResp.Result, nil
}

// UpdateWebhook updates an existing webhook
func (c *Client) UpdateWebhook(id string, wh *Webhook) (*Webhook, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/webhooks/%s", id), wh)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Webhook]
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("API error: %s", apiResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("API request failed")
	}

	return &apiResp.Result, nil
}

// DeleteWebhook deletes a webhook
func (c *Client) DeleteWebhook(id string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/webhooks/%s", id))
	return err
}

// ListWebhooks retrieves all webhooks
func (c *Client) ListWebhooks() ([]Webhook, error) {
	respBody, err := c.Get("/v1/webhooks")
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[Webhook]
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !apiResp.Success {
		if len(apiResp.Errors) > 0 {
			return nil, fmt.Errorf("API error: %s", apiResp.Errors[0].Message)
		}
		return nil, fmt.Errorf("API request failed")
	}

	return apiResp.Result, nil
}
