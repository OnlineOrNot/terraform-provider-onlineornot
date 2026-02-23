package client

import (
	"encoding/json"
	"fmt"
)

// MaintenanceWindow represents a maintenance window
type MaintenanceWindow struct {
	ID              string   `json:"id,omitempty"`
	Name            string   `json:"name"`
	StartDate       string   `json:"start_date"`
	DurationMinutes int      `json:"duration_minutes"`
	DaysOfWeek      []string `json:"days_of_week"`
	Timezone        string   `json:"timezone"`
	Checks          []string `json:"checks,omitempty"`
	Heartbeats      []string `json:"heartbeats,omitempty"`
}

// CreateMaintenanceWindow creates a new maintenance window
func (c *Client) CreateMaintenanceWindow(mw *MaintenanceWindow) (*MaintenanceWindow, error) {
	respBody, err := c.Post("/v1/maintenance-windows", mw)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[MaintenanceWindow]
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

// GetMaintenanceWindow retrieves a maintenance window by ID
func (c *Client) GetMaintenanceWindow(id string) (*MaintenanceWindow, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/maintenance-windows/%s", id))
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[MaintenanceWindow]
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

// UpdateMaintenanceWindow updates an existing maintenance window
func (c *Client) UpdateMaintenanceWindow(id string, mw *MaintenanceWindow) (*MaintenanceWindow, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/maintenance-windows/%s", id), mw)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[MaintenanceWindow]
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

// DeleteMaintenanceWindow deletes a maintenance window
func (c *Client) DeleteMaintenanceWindow(id string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/maintenance-windows/%s", id))
	return err
}

// ListMaintenanceWindows retrieves all maintenance windows
func (c *Client) ListMaintenanceWindows() ([]MaintenanceWindow, error) {
	respBody, err := c.Get("/v1/maintenance-windows")
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[MaintenanceWindow]
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
