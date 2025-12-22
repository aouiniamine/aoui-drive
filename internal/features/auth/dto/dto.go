package dto

type Role string

const (
	RoleAdmin   Role = "ADMIN"
	RoleManager Role = "MANAGER"
	RoleUser    Role = "USER"
)

// Requests

type LoginRequest struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
}

type CreateClientRequest struct {
	Name string `json:"name"`
	Role Role   `json:"role"`
}

// Responses

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

type ClientResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key,omitempty"`
	Role      Role   `json:"role"`
}

type SecretResponse struct {
	SecretKey string `json:"secret_key"`
}
