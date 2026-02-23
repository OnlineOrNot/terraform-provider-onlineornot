package client

import (
	"encoding/json"
	"fmt"
)

// Check represents an uptime check
type Check struct {
	ID                           string            `json:"id,omitempty"`
	Name                         string            `json:"name"`
	URL                          string            `json:"url"`
	CheckType                    string            `json:"check_type,omitempty"`
	Status                       string            `json:"status,omitempty"`
	LastQueued                   string            `json:"last_queued,omitempty"`
	TestInterval                 int               `json:"test_interval,omitempty"`
	TextToSearchFor              string            `json:"text_to_search_for,omitempty"`
	ReminderAlertIntervalMinutes int               `json:"reminder_alert_interval_minutes,omitempty"`
	ConfirmationPeriodSeconds    int               `json:"confirmation_period_seconds,omitempty"`
	RecoveryPeriodSeconds        int               `json:"recovery_period_seconds,omitempty"`
	Timeout                      int               `json:"timeout,omitempty"`
	Method                       string            `json:"method,omitempty"`
	Body                         string            `json:"body,omitempty"`
	Headers                      map[string]string `json:"headers,omitempty"`
	FollowRedirects              *bool             `json:"follow_redirects,omitempty"`
	VerifySSL                    *bool             `json:"verify_ssl,omitempty"`
	AuthUsername                 string            `json:"auth_username,omitempty"`
	AuthPassword                 string            `json:"auth_password,omitempty"`
	AlertPriority                string            `json:"alert_priority,omitempty"`
	Type                         string            `json:"type,omitempty"`
	Version                      string            `json:"version,omitempty"`
	TestRegions                  []string          `json:"test_regions,omitempty"`
	UserAlerts                   []string          `json:"user_alerts,omitempty"`
	SlackAlerts                  []string          `json:"slack_alerts,omitempty"`
	DiscordAlerts                []string          `json:"discord_alerts,omitempty"`
	WebhookAlerts                []string          `json:"webhook_alerts,omitempty"`
	OncallAlerts                 []string          `json:"oncall_alerts,omitempty"`
	IncidentIOAlerts             []string          `json:"incident_io_alerts,omitempty"`
	MicrosoftTeamsAlerts         []string          `json:"microsoft_teams_alerts,omitempty"`
	Assertions                   []Assertion       `json:"assertions,omitempty"`
}

// Assertion represents a check assertion
type Assertion struct {
	Type       string `json:"type"`
	Property   string `json:"property"`
	Comparison string `json:"comparison"`
	Expected   string `json:"expected"`
}

// CreateCheck creates a new uptime check
func (c *Client) CreateCheck(check *Check) (*Check, error) {
	respBody, err := c.Post("/v1/checks", check)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Check]
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

// GetCheck retrieves a check by ID
func (c *Client) GetCheck(id string) (*Check, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/checks/%s", id))
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Check]
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

// UpdateCheck updates an existing check
func (c *Client) UpdateCheck(id string, check *Check) (*Check, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/checks/%s", id), check)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Check]
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

// DeleteCheck deletes a check
func (c *Client) DeleteCheck(id string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/checks/%s", id))
	return err
}

// ListChecks retrieves all checks
func (c *Client) ListChecks() ([]Check, error) {
	respBody, err := c.Get("/v1/checks")
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[Check]
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
