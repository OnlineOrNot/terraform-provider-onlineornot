package client

import (
	"encoding/json"
	"fmt"
)

// StatusPage represents a status page
type StatusPage struct {
	ID                    string   `json:"id,omitempty"`
	Name                  string   `json:"name"`
	Subdomain             string   `json:"subdomain"`
	Description           string   `json:"description,omitempty"`
	CustomDomain          string   `json:"custom_domain,omitempty"`
	Password              string   `json:"password,omitempty"`
	HideFromSearchEngines bool     `json:"hide_from_search_engines,omitempty"`
	AllowedIPs            []string `json:"allowed_ips,omitempty"`
}

// CreateStatusPage creates a new status page
func (c *Client) CreateStatusPage(sp *StatusPage) (*StatusPage, error) {
	respBody, err := c.Post("/v1/status_pages", sp)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPage]
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

// GetStatusPage retrieves a status page by ID
func (c *Client) GetStatusPage(id string) (*StatusPage, error) {
	respBody, err := c.Get(fmt.Sprintf("/v1/status_pages/%s", id))
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPage]
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

// UpdateStatusPage updates an existing status page
func (c *Client) UpdateStatusPage(id string, sp *StatusPage) (*StatusPage, error) {
	respBody, err := c.Patch(fmt.Sprintf("/v1/status_pages/%s", id), sp)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[StatusPage]
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

// DeleteStatusPage deletes a status page
func (c *Client) DeleteStatusPage(id string) error {
	_, err := c.Delete(fmt.Sprintf("/v1/status_pages/%s", id))
	return err
}

// ListStatusPages retrieves all status pages
func (c *Client) ListStatusPages() ([]StatusPage, error) {
	respBody, err := c.Get("/v1/status_pages")
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[StatusPage]
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
