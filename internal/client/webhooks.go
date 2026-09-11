package client

import (
	"encoding/json"
	"fmt"
)

// Webhook is the API response with associations flattened to IDs.
type Webhook struct {
	ID            string   `json:"id,omitempty"`
	URL           string   `json:"url"`
	Description   string   `json:"description,omitempty"`
	Events        []string `json:"events"`
	CheckIDs      []string `json:"-"`
	HeartbeatIDs  []string `json:"-"`
	StatusPageIDs []string `json:"-"`
}

// WebhookRequest uses write-only association names. A nil pointer omits an
// association, while a pointer to an empty slice explicitly clears it on PATCH.
type WebhookRequest struct {
	URL           string    `json:"url,omitempty"`
	Description   *string   `json:"description,omitempty"`
	Events        []string  `json:"events,omitempty"`
	CheckIDs      *[]string `json:"check_ids,omitempty"`
	HeartbeatIDs  *[]string `json:"heartbeat_ids,omitempty"`
	StatusPageIDs *[]string `json:"status_page_ids,omitempty"`
}

// UnmarshalJSON maps the API's association objects to Terraform's ID lists.
// Requests deliberately use WebhookRequest instead of this response model.
func (w *Webhook) UnmarshalJSON(data []byte) error {
	type webhookFields Webhook
	type reference struct {
		ID string `json:"id"`
	}
	var response struct {
		webhookFields
		Checks      []reference `json:"checks"`
		Heartbeats  []reference `json:"heartbeats"`
		StatusPages []reference `json:"status_pages"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}
	ids := func(refs []reference) []string {
		result := make([]string, len(refs))
		for i, ref := range refs {
			result[i] = ref.ID
		}
		return result
	}
	*w = Webhook(response.webhookFields)
	w.CheckIDs = ids(response.Checks)
	w.HeartbeatIDs = ids(response.Heartbeats)
	w.StatusPageIDs = ids(response.StatusPages)
	return nil
}

// CreateWebhook creates a new webhook
func (c *Client) CreateWebhook(wh *WebhookRequest) (*Webhook, error) {
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
func (c *Client) UpdateWebhook(id string, wh *WebhookRequest) (*Webhook, error) {
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

// ListWebhooks retrieves every numbered page, including when the server caps page size.
func (c *Client) ListWebhooks() ([]Webhook, error) {
	return listAll[Webhook](c, "/v1/webhooks")
}
