package client

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// listAll requests numbered pages without assuming the server honors our page size.
// Ordering reads deliberately use their own stricter membership consistency checks.
func listAll[T any](c *Client, path string) ([]T, error) {
	endpoint, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("invalid list path: %w", err)
	}
	query := endpoint.Query()
	query.Set("per_page", "100")
	items := make([]T, 0)
	for page := 1; ; page++ {
		query.Set("page", strconv.Itoa(page))
		endpoint.RawQuery = query.Encode()
		body, err := c.Get(endpoint.String())
		if err != nil {
			return nil, err
		}

		var response struct {
			APIResponse[[]T]
			ResultInfo *struct {
				// Some routers echo raw query strings instead of JSON numbers.
				Page       *json.Number `json:"page"`
				PerPage    *json.Number `json:"per_page"`
				Count      *int         `json:"count"`
				TotalCount *int         `json:"total_count"`
			} `json:"result_info"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		if !response.Success {
			if len(response.Errors) > 0 {
				return nil, fmt.Errorf("API error: %s", response.Errors[0].Message)
			}
			return nil, fmt.Errorf("API request failed")
		}

		info := response.ResultInfo
		// A missing total must not silently turn a paginated response into one page.
		if info == nil || info.TotalCount == nil || *info.TotalCount < 0 {
			return nil, fmt.Errorf("invalid pagination total_count")
		}
		if info.Page != nil {
			responsePage, err := info.Page.Int64()
			if err != nil || responsePage != int64(page) {
				return nil, fmt.Errorf("invalid pagination page: expected %d", page)
			}
		}
		if info.PerPage != nil {
			perPage, err := info.PerPage.Int64()
			if err != nil || perPage <= 0 {
				return nil, fmt.Errorf("invalid pagination per_page")
			}
		}
		if info.Count != nil && *info.Count != len(response.Result) {
			return nil, fmt.Errorf("invalid pagination count")
		}

		items = append(items, response.Result...)
		if len(items) > *info.TotalCount {
			return nil, fmt.Errorf("pagination results exceed total_count")
		}
		if len(response.Result) == 0 || len(items) == *info.TotalCount {
			return items, nil
		}
	}
}
