package client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// EnvironmentVariable contains metadata only. Secret values are never decoded into responses.
type EnvironmentVariable struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type EnvironmentVariableWrite struct {
	Name  string  `json:"name,omitempty"`
	Type  string  `json:"type,omitempty"`
	Value *string `json:"value,omitempty"`
}

func decodeEnvironmentVariable(body []byte) (*EnvironmentVariable, error) {
	var response APIResponse[EnvironmentVariable]
	if json.Unmarshal(body, &response) != nil || !response.Success || response.Result.ID == "" || response.Result.Name == "" || response.Result.Type == "" {
		return nil, fmt.Errorf("invalid environment variable response")
	}
	return &response.Result, nil
}

func (c *Client) CreateEnvironmentVariable(input EnvironmentVariableWrite) (*EnvironmentVariable, error) {
	body, err := c.Post("/v1/env", input)
	if err != nil {
		return nil, err
	}
	return decodeEnvironmentVariable(body)
}
func (c *Client) GetEnvironmentVariable(id string) (*EnvironmentVariable, error) {
	body, err := c.Get("/v1/env/" + url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	return decodeEnvironmentVariable(body)
}
func (c *Client) UpdateEnvironmentVariable(id string, input EnvironmentVariableWrite) (*EnvironmentVariable, error) {
	body, err := c.Patch("/v1/env/"+url.PathEscape(id), input)
	if err != nil {
		return nil, err
	}
	return decodeEnvironmentVariable(body)
}
func (c *Client) DeleteEnvironmentVariable(id string) error {
	body, err := c.Delete("/v1/env/" + url.PathEscape(id))
	if IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var response APIResponse[struct {
		ID string `json:"id"`
	}]
	if json.Unmarshal(body, &response) != nil || !response.Success || response.Result.ID != id {
		return fmt.Errorf("invalid environment variable delete response")
	}
	return nil
}
