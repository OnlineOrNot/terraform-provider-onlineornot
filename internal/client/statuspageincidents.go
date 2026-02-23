package client

import (
	"encoding/json"
	"fmt"
)

// StatusPageIncidentComponent represents a component affected by an incident
type StatusPageIncidentComponent struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// StatusPageIncident represents a status page incident
type StatusPageIncident struct {
	ID                string                        `json:"id,omitempty"`
	StatusPageID      string                        `json:"status_page_id,omitempty"`
	Title             string                        `json:"title"`
	Description       string                        `json:"description"`
	Status            string                        `json:"status"`
	NotifySubscribers *bool                         `json:"notify_subscribers,omitempty"`
	Components        []StatusPageIncidentComponent `json:"components,omitempty"`
}

// CreateStatusPageIncident creates a new status page incident
func (c *Client) CreateStatusPageIncident(statusPageID string, incident *StatusPageIncident) (*StatusPageIncident, error) {
	respBody, err := c.Post(fmt.Sprintf("/v1/status_pages/%s/incidents", statusPageID), incident)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageIncident]
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

// GetStatusPageIncident retrieves a status page incident by ID
func (c *Client) GetStatusPageIncident(statusPageID, incidentID string) (*StatusPageIncident, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/status_pages/%s/incidents/%s", statusPageID, incidentID))
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageIncident]
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

// UpdateStatusPageIncident updates an existing status page incident
func (c *Client) UpdateStatusPageIncident(statusPageID, incidentID string, incident *StatusPageIncident) (*StatusPageIncident, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/status_pages/%s/incidents/%s", statusPageID, incidentID), incident)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageIncident]
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

// DeleteStatusPageIncident deletes a status page incident
func (c *Client) DeleteStatusPageIncident(statusPageID, incidentID string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/status_pages/%s/incidents/%s", statusPageID, incidentID))
	return err
}

// ListStatusPageIncidents retrieves all incidents for a status page
func (c *Client) ListStatusPageIncidents(statusPageID string) ([]StatusPageIncident, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/status_pages/%s/incidents", statusPageID))
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[StatusPageIncident]
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
