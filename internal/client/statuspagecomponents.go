package client

import (
	"encoding/json"
	"fmt"
)

// StatusPageComponent represents a status page component
type StatusPageComponent struct {
	ID             string `json:"id,omitempty"`
	StatusPageID   string `json:"status_page_id,omitempty"`
	Name           string `json:"name"`
	Status         string `json:"status,omitempty"`
	DisplayUptime  *bool  `json:"display_uptime,omitempty"`
	DisplayMetrics *bool  `json:"display_metrics,omitempty"`
}

// CreateStatusPageComponent creates a new status page component
func (c *Client) CreateStatusPageComponent(statusPageID string, comp *StatusPageComponent) (*StatusPageComponent, error) {
	respBody, err := c.Post(fmt.Sprintf("/v1/status_pages/%s/components", statusPageID), comp)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageComponent]
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

// GetStatusPageComponent retrieves a status page component by ID
func (c *Client) GetStatusPageComponent(statusPageID, componentID string) (*StatusPageComponent, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/status_pages/%s/components/%s", statusPageID, componentID))
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageComponent]
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

// UpdateStatusPageComponent updates an existing status page component
func (c *Client) UpdateStatusPageComponent(statusPageID, componentID string, comp *StatusPageComponent) (*StatusPageComponent, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/status_pages/%s/components/%s", statusPageID, componentID), comp)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageComponent]
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

// DeleteStatusPageComponent deletes a status page component
func (c *Client) DeleteStatusPageComponent(statusPageID, componentID string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/status_pages/%s/components/%s", statusPageID, componentID))
	return err
}

// ListStatusPageComponents retrieves all components for a status page
func (c *Client) ListStatusPageComponents(statusPageID string) ([]StatusPageComponent, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/status_pages/%s/components", statusPageID))
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[StatusPageComponent]
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
