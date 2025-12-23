-- Webhook URLs queries

-- name: GetWebhookURLByID :one
SELECT id, bucket_id, url, event_type, is_active, created_at, updated_at
FROM webhook_urls WHERE id = ?;

-- name: ListWebhookURLsByBucketID :many
SELECT id, bucket_id, url, event_type, is_active, created_at, updated_at
FROM webhook_urls WHERE bucket_id = ? ORDER BY created_at DESC;

-- name: ListActiveWebhookURLsByBucketAndEvent :many
SELECT id, bucket_id, url, event_type, is_active, created_at, updated_at
FROM webhook_urls WHERE bucket_id = ? AND event_type = ? AND is_active = 1;

-- name: CreateWebhookURL :one
INSERT INTO webhook_urls (id, bucket_id, url, event_type, is_active)
VALUES (?, ?, ?, ?, ?)
RETURNING id, bucket_id, url, event_type, is_active, created_at, updated_at;

-- name: UpdateWebhookURL :one
UPDATE webhook_urls
SET url = ?, event_type = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, bucket_id, url, event_type, is_active, created_at, updated_at;

-- name: DeleteWebhookURL :execrows
DELETE FROM webhook_urls WHERE id = ?;

-- name: WebhookURLExists :one
SELECT EXISTS(SELECT 1 FROM webhook_urls WHERE bucket_id = ? AND url = ? AND event_type = ?) AS webhook_exists;

-- Webhook Headers queries

-- name: GetWebhookHeaderByID :one
SELECT id, webhook_url_id, header_name, header_value, created_at
FROM webhook_headers WHERE id = ?;

-- name: ListWebhookHeadersByURLID :many
SELECT id, webhook_url_id, header_name, header_value, created_at
FROM webhook_headers WHERE webhook_url_id = ? ORDER BY header_name;

-- name: CreateWebhookHeader :one
INSERT INTO webhook_headers (id, webhook_url_id, header_name, header_value)
VALUES (?, ?, ?, ?)
RETURNING id, webhook_url_id, header_name, header_value, created_at;

-- name: UpdateWebhookHeader :one
UPDATE webhook_headers SET header_value = ? WHERE id = ?
RETURNING id, webhook_url_id, header_name, header_value, created_at;

-- name: DeleteWebhookHeader :execrows
DELETE FROM webhook_headers WHERE id = ?;

-- name: DeleteWebhookHeadersByURLID :execrows
DELETE FROM webhook_headers WHERE webhook_url_id = ?;

-- Webhook Events queries

-- name: GetWebhookEventByID :one
SELECT id, webhook_url_id, bucket_id, resource_id, event_type, status, payload,
       response_code, response_body, attempts, max_attempts, next_retry_at,
       last_attempt_at, created_at, completed_at
FROM webhook_events WHERE id = ?;

-- name: ListWebhookEventsByBucketID :many
SELECT id, webhook_url_id, bucket_id, resource_id, event_type, status, payload,
       response_code, response_body, attempts, max_attempts, next_retry_at,
       last_attempt_at, created_at, completed_at
FROM webhook_events WHERE bucket_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListPendingWebhookEvents :many
SELECT id, webhook_url_id, bucket_id, resource_id, event_type, status, payload,
       response_code, response_body, attempts, max_attempts, next_retry_at,
       last_attempt_at, created_at, completed_at
FROM webhook_events
WHERE (status = 'pending' OR (status = 'retrying' AND next_retry_at <= CURRENT_TIMESTAMP))
AND attempts < max_attempts
ORDER BY created_at ASC LIMIT ?;

-- name: CreateWebhookEvent :one
INSERT INTO webhook_events (id, webhook_url_id, bucket_id, resource_id, event_type, status, payload, max_attempts)
VALUES (?, ?, ?, ?, ?, 'pending', ?, ?)
RETURNING id, webhook_url_id, bucket_id, resource_id, event_type, status, payload,
          response_code, response_body, attempts, max_attempts, next_retry_at,
          last_attempt_at, created_at, completed_at;

-- name: UpdateWebhookEventStatus :exec
UPDATE webhook_events
SET status = ?, response_code = ?, response_body = ?, attempts = attempts + 1,
    last_attempt_at = CURRENT_TIMESTAMP, next_retry_at = ?, completed_at = ?
WHERE id = ?;

-- name: CountWebhookEventsByBucketID :one
SELECT COUNT(*) AS count FROM webhook_events WHERE bucket_id = ?;
