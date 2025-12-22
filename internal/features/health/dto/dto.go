package dto

type HealthResponse struct {
	Status string `json:"status"`
}

type ReadyResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}
