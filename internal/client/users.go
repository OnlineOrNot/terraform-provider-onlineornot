package client

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
	return listAll[User](c, "/v1/users")
}
