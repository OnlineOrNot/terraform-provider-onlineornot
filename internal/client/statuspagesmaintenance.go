package client

import (
	"encoding/json"
	"fmt"
)

// ScheduledMaintenanceNotifications represents notification settings
type ScheduledMaintenanceNotifications struct {
	AnHourBefore bool `json:"an_hour_before"`
	AtStart      bool `json:"at_start"`
	AtEnd        bool `json:"at_end"`
}

// StatusPageScheduledMaintenance represents a status page scheduled maintenance
type StatusPageScheduledMaintenance struct {
	ID                 string                             `json:"id,omitempty"`
	StatusPageID       string                             `json:"status_page_id,omitempty"`
	Title              string                             `json:"title"`
	Description        string                             `json:"description"`
	StartDate          string                             `json:"start_date"`
	DurationMinutes    int                                `json:"duration_minutes"`
	ComponentsAffected []string                           `json:"components_affected,omitempty"`
	Notifications      *ScheduledMaintenanceNotifications `json:"notifications,omitempty"`
}

// CreateStatusPageScheduledMaintenance creates a new scheduled maintenance
func (c *Client) CreateStatusPageScheduledMaintenance(statusPageID string, sm *StatusPageScheduledMaintenance) (*StatusPageScheduledMaintenance, error) {
	respBody, err := c.Post(fmt.Sprintf("/v1/status_pages/%s/scheduled_maintenance", statusPageID), sm)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageScheduledMaintenance]
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

// GetStatusPageScheduledMaintenance retrieves a scheduled maintenance by ID
func (c *Client) GetStatusPageScheduledMaintenance(statusPageID, smID string) (*StatusPageScheduledMaintenance, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/status_pages/%s/scheduled_maintenance/%s", statusPageID, smID))
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageScheduledMaintenance]
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

// UpdateStatusPageScheduledMaintenance updates an existing scheduled maintenance
func (c *Client) UpdateStatusPageScheduledMaintenance(statusPageID, smID string, sm *StatusPageScheduledMaintenance) (*StatusPageScheduledMaintenance, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/status_pages/%s/scheduled_maintenance/%s", statusPageID, smID), sm)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPageScheduledMaintenance]
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

// DeleteStatusPageScheduledMaintenance deletes a scheduled maintenance
func (c *Client) DeleteStatusPageScheduledMaintenance(statusPageID, smID string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/status_pages/%s/scheduled_maintenance/%s", statusPageID, smID))
	return err
}

// ListStatusPageScheduledMaintenances retrieves all scheduled maintenances for a status page
func (c *Client) ListStatusPageScheduledMaintenances(statusPageID string) ([]StatusPageScheduledMaintenance, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/status_pages/%s/scheduled_maintenance", statusPageID))
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[StatusPageScheduledMaintenance]
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
