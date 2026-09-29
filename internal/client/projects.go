package client

import (
	"fmt"
	"net/url"
	"regexp"
)

var projectIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{16,128}$`)
var projectIDLetter = regexp.MustCompile(`[A-Za-z-]`)

// ValidProjectID checks syntax only; the API checks table identity and ownership.
func ValidProjectID(id string) bool {
	return projectIDPattern.MatchString(id) && projectIDLetter.MatchString(id)
}
func projectPath(id string) (string, error) {
	if !ValidProjectID(id) {
		return "", fmt.Errorf("expected an encoded project ID")
	}
	return "/v1/projects/" + url.PathEscape(id), nil
}

type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (c *Client) CreateProject(name string) (*Project, error) {
	body, err := c.Post("/v1/projects", map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[Project](body)
}
func (c *Client) GetProject(id string) (*Project, error) {
	path, err := projectPath(id)
	if err != nil {
		return nil, err
	}
	body, err := c.Get(path)
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[Project](body)
}
func (c *Client) UpdateProject(id, name string) (*Project, error) {
	path, err := projectPath(id)
	if err != nil {
		return nil, err
	}
	body, err := c.Patch(path, map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	return parseAPIResponse[Project](body)
}
func (c *Client) DeleteProject(id string) error {
	path, err := projectPath(id)
	if err != nil {
		return err
	}
	body, err := c.Delete(path)
	if IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	result, err := parseAPIResponse[Project](body)
	if err != nil {
		return err
	}
	if result.ID != id {
		return fmt.Errorf("invalid project deletion acknowledgement")
	}
	return nil
}
func (c *Client) ListProjects() ([]Project, error)   { return listAll[Project](c, "/v1/projects") }
func (c *Client) MoveCheck(id, project string) error { return c.moveResource("checks", id, project) }
func (c *Client) MoveHeartbeat(id, project string) error {
	return c.moveResource("heartbeats", id, project)
}
func (c *Client) moveResource(kind, id, project string) error {
	if !ValidProjectID(project) {
		return fmt.Errorf("expected an encoded project ID")
	}
	body, err := c.Post("/v1/"+kind+"/"+url.PathEscape(id)+"/move", map[string]string{"project_id": project})
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
	if result.ID != id || result.ProjectID != project {
		return fmt.Errorf("invalid project move acknowledgement")
	}
	return nil
}
func projectFilter(path, project string) (string, error) {
	if project == "" {
		return path, nil
	}
	if !ValidProjectID(project) {
		return "", fmt.Errorf("expected an encoded project ID")
	}
	return path + "?project_id=" + url.QueryEscape(project), nil
}
