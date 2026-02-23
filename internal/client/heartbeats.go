package client

import (
	"encoding/json"
	"fmt"
)

// Heartbeat represents a heartbeat monitor
type Heartbeat struct {
	ID                           string   `json:"id,omitempty"`
	Name                         string   `json:"name"`
	ReportPeriod                 int      `json:"report_period,omitempty"`
	ReportPeriodCron             string   `json:"report_period_cron,omitempty"`
	GracePeriod                  int      `json:"grace_period"`
	Timezone                     string   `json:"timezone,omitempty"`
	AlertPriority                string   `json:"alert_priority,omitempty"`
	ReminderAlertIntervalMinutes int      `json:"reminder_alert_interval_minutes,omitempty"`
	UserAlerts                   []string `json:"user_alerts,omitempty"`
	SlackAlerts                  []string `json:"slack_alerts,omitempty"`
	DiscordAlerts                []string `json:"discord_alerts,omitempty"`
	WebhookAlerts                []string `json:"webhook_alerts,omitempty"`
	OncallAlerts                 []string `json:"oncall_alerts,omitempty"`
	IncidentIOAlerts             []string `json:"incident_io_alerts,omitempty"`
	MicrosoftTeamsAlerts         []string `json:"microsoft_teams_alerts,omitempty"`
}

// CreateHeartbeat creates a new heartbeat
func (c *Client) CreateHeartbeat(hb *Heartbeat) (*Heartbeat, error) {
	respBody, err := c.Post("/v1/heartbeats", hb)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Heartbeat]
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

// GetHeartbeat retrieves a heartbeat by ID
func (c *Client) GetHeartbeat(id string) (*Heartbeat, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/heartbeats/%s", id))
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Heartbeat]
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

// UpdateHeartbeat updates an existing heartbeat
func (c *Client) UpdateHeartbeat(id string, hb *Heartbeat) (*Heartbeat, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/heartbeats/%s", id), hb)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Heartbeat]
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

// DeleteHeartbeat deletes a heartbeat
func (c *Client) DeleteHeartbeat(id string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/heartbeats/%s", id))
	return err
}

// ListHeartbeats retrieves all heartbeats
func (c *Client) ListHeartbeats() ([]Heartbeat, error) {
	respBody, err := c.Get("/v1/heartbeats")
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[Heartbeat]
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
