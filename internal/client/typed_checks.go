package client

import (
	"encoding/json"
	"fmt"
)

type MonitorAssertion struct {
	Type       string `json:"type"`
	Property   string `json:"property"`
	Comparison string `json:"comparison"`
	Expected   string `json:"expected"`
}

type DNSCheck struct {
	ID                           string             `json:"id,omitempty"`
	Name                         string             `json:"name"`
	CheckType                    string             `json:"check_type,omitempty"`
	Status                       string             `json:"status,omitempty"`
	LastQueued                   string             `json:"last_queued,omitempty"`
	TestInterval                 int                `json:"test_interval,omitempty"`
	ReminderAlertIntervalMinutes int                `json:"reminder_alert_interval_minutes,omitempty"`
	ConfirmationPeriodSeconds    int                `json:"confirmation_period_seconds,omitempty"`
	RecoveryPeriodSeconds        int                `json:"recovery_period_seconds,omitempty"`
	Timeout                      int                `json:"timeout,omitempty"`
	AlertPriority                string             `json:"alert_priority,omitempty"`
	DNSDomain                    string             `json:"dns_domain"`
	DNSRecordType                string             `json:"dns_record_type"`
	DNSResolver                  *string            `json:"dns_resolver,omitempty"`
	DNSProtocol                  string             `json:"dns_protocol,omitempty"`
	TestRegions                  []string           `json:"test_regions,omitempty"`
	UserAlerts                   []string           `json:"user_alerts,omitempty"`
	SlackAlerts                  []string           `json:"slack_alerts,omitempty"`
	DiscordAlerts                []string           `json:"discord_alerts,omitempty"`
	TelegramAlerts               []string           `json:"telegram_alerts,omitempty"`
	WebhookAlerts                []string           `json:"webhook_alerts,omitempty"`
	OncallAlerts                 []string           `json:"oncall_alerts,omitempty"`
	IncidentIOAlerts             []string           `json:"incident_io_alerts,omitempty"`
	MicrosoftTeamsAlerts         []string           `json:"microsoft_teams_alerts,omitempty"`
	Assertions                   []MonitorAssertion `json:"assertions,omitempty"`
}

type TCPCheck struct {
	ID                           string             `json:"id,omitempty"`
	Name                         string             `json:"name"`
	CheckType                    string             `json:"check_type,omitempty"`
	Status                       string             `json:"status,omitempty"`
	LastQueued                   string             `json:"last_queued,omitempty"`
	TestInterval                 int                `json:"test_interval,omitempty"`
	ReminderAlertIntervalMinutes int                `json:"reminder_alert_interval_minutes,omitempty"`
	ConfirmationPeriodSeconds    int                `json:"confirmation_period_seconds,omitempty"`
	RecoveryPeriodSeconds        int                `json:"recovery_period_seconds,omitempty"`
	Timeout                      int                `json:"timeout,omitempty"`
	AlertPriority                string             `json:"alert_priority,omitempty"`
	TCPHostname                  string             `json:"tcp_hostname"`
	TCPPort                      int                `json:"tcp_port"`
	TCPIPFamily                  string             `json:"tcp_ip_family,omitempty"`
	TCPData                      *string            `json:"tcp_data,omitempty"`
	TCPShouldFail                *bool              `json:"tcp_should_fail,omitempty"`
	TestRegions                  []string           `json:"test_regions,omitempty"`
	UserAlerts                   []string           `json:"user_alerts,omitempty"`
	SlackAlerts                  []string           `json:"slack_alerts,omitempty"`
	DiscordAlerts                []string           `json:"discord_alerts,omitempty"`
	TelegramAlerts               []string           `json:"telegram_alerts,omitempty"`
	WebhookAlerts                []string           `json:"webhook_alerts,omitempty"`
	OncallAlerts                 []string           `json:"oncall_alerts,omitempty"`
	IncidentIOAlerts             []string           `json:"incident_io_alerts,omitempty"`
	MicrosoftTeamsAlerts         []string           `json:"microsoft_teams_alerts,omitempty"`
	Assertions                   []MonitorAssertion `json:"assertions,omitempty"`
}

func parseAPIResponse[T any](respBody []byte) (*T, error) {
	var apiResp APIResponse[T]
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

func (c *Client) CreateDNSCheck(check *DNSCheck) (*DNSCheck, error) {
	respBody, err := c.Post("/v1/checks/dns", check)
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[DNSCheck](respBody)
}

func (c *Client) GetDNSCheck(id string) (*DNSCheck, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/checks/dns/%s", id))
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[DNSCheck](respBody)
}

func (c *Client) UpdateDNSCheck(id string, check *DNSCheck) (*DNSCheck, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/checks/dns/%s", id), check)
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[DNSCheck](respBody)
}

func (c *Client) DeleteDNSCheck(id string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/checks/dns/%s", id))
	return err
}

func (c *Client) CreateTCPCheck(check *TCPCheck) (*TCPCheck, error) {
	respBody, err := c.Post("/v1/checks/tcp", check)
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[TCPCheck](respBody)
}

func (c *Client) GetTCPCheck(id string) (*TCPCheck, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/checks/tcp/%s", id))
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[TCPCheck](respBody)
}

func (c *Client) UpdateTCPCheck(id string, check *TCPCheck) (*TCPCheck, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/checks/tcp/%s", id), check)
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[TCPCheck](respBody)
}

func (c *Client) DeleteTCPCheck(id string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/checks/tcp/%s", id))
	return err
}
