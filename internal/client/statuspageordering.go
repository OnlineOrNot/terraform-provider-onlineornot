package client

import (
	"encoding/json"
	"fmt"
	"regexp"
)

var orderingID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// ValidOrderingID rejects URL delimiters and traversal, including encoded forms.
func ValidOrderingID(id string) bool { return orderingID.MatchString(id) }

// StatusPageOrderScope identifies one independently ordered collection.
// Kind is components (ungrouped), groups, or group_components.
type StatusPageOrderScope struct{ PageID, GroupID, Kind string }

func (s StatusPageOrderScope) paths() (string, string, error) {
	if !ValidOrderingID(s.PageID) {
		return "", "", fmt.Errorf("invalid status page ID")
	}
	base := "/v1/status_pages/" + s.PageID
	switch s.Kind {
	case "components":
		if s.GroupID != "" {
			break
		}
		return base + "/components", base + "/components/sort-order", nil
	case "groups":
		if s.GroupID != "" {
			break
		}
		return base + "/groups", base + "/groups/sort-order", nil
	case "group_components":
		if !ValidOrderingID(s.GroupID) {
			break
		}
		return base + "/components", base + "/groups/" + s.GroupID + "/sort-order", nil
	}
	return "", "", fmt.Errorf("invalid ordering scope")
}

// CheckStatusPageOrderParent distinguishes an absent parent from an empty list.
func (c *Client) CheckStatusPageOrderParent(s StatusPageOrderScope) error {
	if _, _, err := s.paths(); err != nil {
		return err
	}
	if _, err := c.GetStatusPage(s.PageID); err != nil {
		return err
	}
	if s.Kind == "group_components" {
		_, err := c.GetStatusPageComponentGroup(s.PageID, s.GroupID)
		return err
	}
	return nil
}

// ListStatusPageOrder returns complete scope membership in API response order.
// Pagination is checked before any caller can submit a destructive rank reset.
func (c *Client) ListStatusPageOrder(s StatusPageOrderScope) ([]string, error) {
	listPath, _, err := s.paths()
	if err != nil {
		return nil, err
	}
	ids := []string{}
	seen := map[string]bool{}
	total := -1
	for page := 1; ; page++ {
		body, err := c.Get(fmt.Sprintf("%s?page=%d&per_page=100", listPath, page))
		if err != nil {
			return nil, err
		}
		var result struct {
			Result []struct {
				ID      string  `json:"id"`
				GroupID *string `json:"group_id"`
			} `json:"result"`
			ResultInfo *struct {
				Page       int  `json:"page"`
				Count      *int `json:"count"`
				TotalCount *int `json:"total_count"`
			} `json:"result_info"`
			Success bool       `json:"success"`
			Errors  []APIError `json:"errors"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("invalid ordering list: %w", err)
		}
		if !result.Success {
			return nil, fmt.Errorf("ordering list failed: %v", result.Errors)
		}
		info := result.ResultInfo
		if info == nil || info.Count == nil || info.TotalCount == nil || *info.Count != len(result.Result) || *info.TotalCount < 0 || (info.Page != 0 && info.Page != page) {
			return nil, fmt.Errorf("invalid ordering pagination metadata")
		}
		if total < 0 {
			total = *info.TotalCount
		} else if total != *info.TotalCount {
			return nil, fmt.Errorf("ordering membership changed during pagination; retry")
		}
		for _, item := range result.Result {
			if !ValidOrderingID(item.ID) || seen[item.ID] {
				return nil, fmt.Errorf("invalid or repeated ID in ordering list; retry")
			}
			seen[item.ID] = true
			group := ""
			if item.GroupID != nil {
				group = *item.GroupID
			}
			if s.Kind == "groups" || (s.Kind == "components" && group == "") || (s.Kind == "group_components" && group == s.GroupID) {
				ids = append(ids, item.ID)
			}
		}
		if len(seen) > total {
			return nil, fmt.Errorf("ordering list exceeds total count")
		}
		if len(seen) == total {
			return ids, nil
		}
		if len(result.Result) == 0 {
			return nil, fmt.Errorf("ordering pagination made no progress")
		}
	}
}

// SetStatusPageOrder changes ranks only; it never changes group membership.
func (c *Client) SetStatusPageOrder(s StatusPageOrderScope, ids []string) error {
	_, putPath, err := s.paths()
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !ValidOrderingID(id) || seen[id] {
			return fmt.Errorf("invalid or duplicate ordering ID")
		}
		seen[id] = true
	}
	if ids == nil {
		ids = []string{}
	}
	key := "component_ids"
	if s.Kind == "groups" {
		key = "group_ids"
	}
	body, err := c.Put(putPath, map[string][]string{key: ids})
	if err != nil {
		return err
	}
	var result APIResponse[json.RawMessage]
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("invalid ordering response: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("ordering update failed: %v", result.Errors)
	}
	return nil
}
