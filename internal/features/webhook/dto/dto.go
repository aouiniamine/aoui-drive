package dto

import "time"

// Event types
const (
	EventResourceNew     = "resource.new"
	EventResourceDeleted = "resource.deleted"
)

// Status constants
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusSuccess    = "success"
	StatusFailed     = "failed"
	StatusRetrying   = "retrying"
)

// Requests

type CreateWebhookURLRequest struct {
	URL       string                `json:"url"`
	EventType string                `json:"event_type"`
	IsActive  bool                  `json:"is_active"`
	Headers   []CreateHeaderRequest `json:"headers,omitempty"`
}

type UpdateWebhookURLRequest struct {
	URL       string `json:"url"`
	EventType string `json:"event_type"`
	IsActive  bool   `json:"is_active"`
}

type CreateHeaderRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type UpdateHeaderRequest struct {
	Value string `json:"value"`
}

// Responses

type WebhookURLResponse struct {
	ID        string           `json:"id"`
	BucketID  string           `json:"bucket_id"`
	URL       string           `json:"url"`
	EventType string           `json:"event_type"`
	IsActive  bool             `json:"is_active"`
	Headers   []HeaderResponse `json:"headers,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type HeaderResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
}

type WebhookURLListResponse struct {
	Webhooks []WebhookURLResponse `json:"webhooks"`
}

type WebhookEventResponse struct {
	ID           string     `json:"id"`
	WebhookURLID string     `json:"webhook_url_id"`
	BucketID     string     `json:"bucket_id"`
	ResourceID   string     `json:"resource_id"`
	EventType    string     `json:"event_type"`
	Status       string     `json:"status"`
	ResponseCode *int64     `json:"response_code,omitempty"`
	Attempts     int64      `json:"attempts"`
	MaxAttempts  int64      `json:"max_attempts"`
	NextRetryAt  *time.Time `json:"next_retry_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

type WebhookEventListResponse struct {
	Events []WebhookEventResponse `json:"events"`
	Total  int64                  `json:"total"`
	Page   int                    `json:"page"`
	Limit  int                    `json:"limit"`
}

// Webhook Payload (sent to external URLs)

type WebhookPayload struct {
	Event       string          `json:"event"`
	Timestamp   time.Time       `json:"timestamp"`
	BucketID    string          `json:"bucket_id"`
	BucketName  string          `json:"bucket_name"`
	ResourceID  string          `json:"resource_id"`
	ResourceURL string          `json:"resource_url"`
	Resource    ResourcePayload `json:"resource"`
}

type ResourcePayload struct {
	Hash        string `json:"hash"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Extension   string `json:"extension"`
}
