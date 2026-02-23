package client

import (
	"encoding/json"
	"fmt"
)

// StatusPageComponentGroup represents a status page component group
type StatusPageComponentGroup struct {
	ID           string `json:"id,omitempty"`
	StatusPageID string `json:"status_page_id,omitempty"`
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
}

// CreateStatusPageComponentGroup creates a new status page component group
func (c *Client) CreateStatusPageComponentGroup(statusPageID string, group *StatusPageComponentGroup) (*StatusPageComponentGroup, error) {
	respBody, err := c.Post(fmt.Sprintf("/v1/status_pages/%s/component_groups", statusPageID), group)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageComponentGroup]
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

// GetStatusPageComponentGroup retrieves a status page component group by ID
func (c *Client) GetStatusPageComponentGroup(statusPageID, groupID string) (*StatusPageComponentGroup, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/status_pages/%s/component_groups/%s", statusPageID, groupID))
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageComponentGroup]
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

// UpdateStatusPageComponentGroup updates an existing status page component group
func (c *Client) UpdateStatusPageComponentGroup(statusPageID, groupID string, group *StatusPageComponentGroup) (*StatusPageComponentGroup, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/status_pages/%s/component_groups/%s", statusPageID, groupID), group)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageComponentGroup]
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

// DeleteStatusPageComponentGroup deletes a status page component group
func (c *Client) DeleteStatusPageComponentGroup(statusPageID, groupID string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/status_pages/%s/component_groups/%s", statusPageID, groupID))
	return err
}

// ListStatusPageComponentGroups retrieves all component groups for a status page
func (c *Client) ListStatusPageComponentGroups(statusPageID string) ([]StatusPageComponentGroup, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/status_pages/%s/component_groups", statusPageID))
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[StatusPageComponentGroup]
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
