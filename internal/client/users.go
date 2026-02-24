package client

import (
	"encoding/json"
	"fmt"
)

// User represents a user in the organisation
type User struct {
	ID        string  `json:"id"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Email     *string `json:"email"`
	Image     *string `json:"image"`
	Role      string  `json:"role"`
}

// ListUsers retrieves all users in the organisation
func (c *Client) ListUsers() ([]User, error) {
	respBody, err := c.Get("/v1/users")
	if err != nil {
		return nil, err
	}

	var apiResp APIListResponse[User]
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
