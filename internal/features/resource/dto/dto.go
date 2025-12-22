package dto

import "time"

// Responses

type ResourceResponse struct {
	ID          string    `json:"id"`
	Hash        string    `json:"hash"`
	Size        int64     `json:"size"`
	ContentType string    `json:"content_type"`
	CreatedAt   time.Time `json:"created_at"`
}

type ResourceListResponse struct {
	Resources []ResourceResponse `json:"resources"`
}
