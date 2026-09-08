package client

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// TokenGrant is a scope/permission pair. Token endpoints use camelCase for dates.
type TokenGrant struct {
	Scope      string `json:"scope"`
	Permission string `json:"permission"`
}

type CreateTokenRequest struct {
	Name   string       `json:"name"`
	Grants []TokenGrant `json:"grants"`
	// Nil omits expiry (API default); a pointer to nil sends explicit null.
	ExpiresAt **string `json:"expiresAt,omitempty"`
}

type Token struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Grants       []TokenGrant `json:"grants"`
	ExpiresAfter *string      `json:"expiresAfter"`
	Token        string       `json:"token"`
}

func decodeToken(body []byte) (*Token, error) {
	var resp APIResponse[Token]
	if err := json.Unmarshal(body, &resp); err != nil {
		// JSON errors may contain the secret.
		return nil, fmt.Errorf("invalid token response")
	}
	if !resp.Success || resp.Result.ID == "" {
		return nil, fmt.Errorf("unsuccessful or incomplete token response")
	}
	return &resp.Result, nil
}

func (c *Client) CreateToken(input *CreateTokenRequest) (*Token, error) {
	body, err := c.Post("/v1/tokens", input)
	if err != nil {
		return nil, err
	}
	token, err := decodeToken(body)
	if err == nil && token.Token == "" {
		return nil, fmt.Errorf("create token response did not contain a secret")
	}
	return token, err
}

func (c *Client) GetToken(id string) (*Token, error) {
	body, err := c.Get("/v1/tokens/" + url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	return decodeToken(body)
}

func (c *Client) DeleteToken(id string) error {
	body, err := c.Delete("/v1/tokens/" + url.PathEscape(id))
	if IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var resp APIResponse[struct {
		Deleted bool `json:"deleted"`
	}]
	if json.Unmarshal(body, &resp) != nil || !resp.Success || !resp.Result.Deleted {
		return fmt.Errorf("unsuccessful or invalid delete token response")
	}
	return nil
}
