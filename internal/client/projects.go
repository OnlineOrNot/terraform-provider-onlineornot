package client

import (
	"fmt"
	"net/url"
)

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ProjectWrite struct {
	Name string `json:"name"`
}

func (c *Client) CreateProject(name string) (*Project, error) {
	body, err := c.Post("/v1/projects", ProjectWrite{Name: name})
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[Project](body)
}

func (c *Client) GetProject(id string) (*Project, error) {
	body, err := c.Get("/v1/projects/" + url.PathEscape(id))
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[Project](body)
}

func (c *Client) UpdateProject(id, name string) (*Project, error) {
	body, err := c.Patch("/v1/projects/"+url.PathEscape(id), ProjectWrite{Name: name})
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[Project](body)
}

func (c *Client) DeleteProject(id string) error {
	body, err := c.Delete("/v1/projects/" + url.PathEscape(id))
	if IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	result, err := parseAPIResponse[struct {
		ID string `json:"id"`
	}](body)
	if err != nil {
		return err
	}
	if result.ID != id {
		return fmt.Errorf("invalid project deletion acknowledgement")
	}
	return nil
}

func (c *Client) ListProjects() ([]Project, error) { return listAll[Project](c, "/v1/projects") }

func (c *Client) MoveCheck(id, projectID string) error {
	return c.moveMonitorProject("/v1/checks", id, projectID)
}

func (c *Client) MoveHeartbeat(id, projectID string) error {
	return c.moveMonitorProject("/v1/heartbeats", id, projectID)
}

func (c *Client) moveMonitorProject(path, id, projectID string) error {
	body, err := c.Post(path+"/"+url.PathEscape(id)+"/move", struct {
		ProjectID string `json:"project_id"`
	}{projectID})
	if err != nil {
		return err
	}
	result, err := parseAPIResponse[struct {
		ID        string `json:"id"`
		ProjectID string `json:"project_id"`
	}](body)
	if err != nil {
		return err
	}
	if result.ID != id || result.ProjectID != projectID {
		return fmt.Errorf("invalid project move acknowledgement")
	}
	return nil
}

func projectCollectionPath(path string, projectID string) string {
	if projectID == "" {
		return path
	}
	return path + "?" + url.Values{"project_id": {projectID}}.Encode()
}
