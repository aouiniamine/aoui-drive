package service

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/aouiniamine/aoui-drive/internal/database/sqlc"
	"github.com/aouiniamine/aoui-drive/internal/features/webhook/repository"
)

const (
	requestTimeout = 10 * time.Second
)

// WebhookSender handles sending webhooks directly
type WebhookSender struct {
	repo       repository.WebhookRepository
	httpClient *http.Client
}

func NewWebhookSender(repo repository.WebhookRepository) *WebhookSender {
	return &WebhookSender{
		repo: repo,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// SendWebhook sends a webhook to the specified URL with headers
// extraHeaders are optional headers passed at request time (e.g., from resource upload)
func (s *WebhookSender) SendWebhook(ctx context.Context, webhook *sqlc.WebhookUrl, payload string, extraHeaders map[string]string) error {
	// Get headers for this webhook
	headers, err := s.repo.ListHeadersByURLID(ctx, webhook.ID)
	if err != nil {
		log.Printf("Error fetching webhook headers: %v", err)
		// Continue without custom headers
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook.Url, bytes.NewBufferString(payload))
	if err != nil {
		return err
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "AOUI-Drive-Webhook/1.0")
	req.Header.Set("X-Webhook-Event", webhook.EventType)

	// Add custom headers from webhook configuration
	for _, h := range headers {
		req.Header.Set(h.HeaderName, h.HeaderValue)
	}

	// Add extra headers passed at request time (these take precedence)
	for name, value := range extraHeaders {
		req.Header.Set(name, value)
	}

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		log.Printf("Webhook delivery failed for %s: %v", webhook.Url, err)
		return err
	}
	defer resp.Body.Close()

	// Read and discard response body
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Webhook delivered successfully to %s (status: %d)", webhook.Url, resp.StatusCode)
	} else {
		log.Printf("Webhook delivery failed for %s (status: %d)", webhook.Url, resp.StatusCode)
	}

	return nil
}
