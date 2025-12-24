# Webhook Integration Architecture

## Overview

AOUI Drive supports webhook notifications that are triggered when resources are created or deleted in a bucket. Webhooks allow external systems to receive real-time notifications about storage events.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Resource       │────▶│  Webhook         │────▶│  External       │
│  Service        │     │  Service         │     │  Endpoint       │
│                 │     │                  │     │                 │
│  - Upload       │     │  - TriggerEvent  │     │  HTTP POST      │
│  - Delete       │     │  - SendWebhook   │     │  + Headers      │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

### Components

1. **Resource Service** - Triggers webhook events on resource operations
2. **Webhook Service** - Manages webhook URLs, headers, and dispatches HTTP requests
3. **Webhook Sender** - Handles HTTP delivery to external endpoints

## Database Schema

### Tables

```sql
-- Webhook URLs per bucket
webhook_urls
├── id              TEXT PRIMARY KEY
├── bucket_id       TEXT NOT NULL (FK → buckets.id)
├── url             TEXT NOT NULL
├── event_type      TEXT NOT NULL ('resource.new' | 'resource.deleted')
├── is_active       INTEGER DEFAULT 1
├── created_at      DATETIME
└── updated_at      DATETIME

-- Custom headers for webhook requests
webhook_headers
├── id              TEXT PRIMARY KEY
├── webhook_url_id  TEXT NOT NULL (FK → webhook_urls.id)
├── header_name     TEXT NOT NULL
├── header_value    TEXT NOT NULL
└── created_at      DATETIME
```

## Event Types

| Event Type         | Trigger                          |
|--------------------|----------------------------------|
| `resource.new`     | When a new resource is uploaded  |
| `resource.deleted` | When a resource is deleted       |

## Payload Format

Webhooks are sent as HTTP POST requests with `Content-Type: application/json`:

```json
{
  "event": "resource.new",
  "timestamp": "2025-12-23T10:30:00Z",
  "bucket_id": "550e8400-e29b-41d4-a716-446655440000",
  "bucket_name": "my-bucket",
  "resource_id": "660e8400-e29b-41d4-a716-446655440001",
  "resource_url": "https://example.com/resources/550e8400.../abc123def456/download",
  "resource": {
    "hash": "abc123def456...",
    "size": 12345,
    "content_type": "image/png",
    "extension": ".png"
  }
}
```

> **Note:** The `resource_url` uses the download endpoint (`/resources/{bucketId}/{hash}/download`) which works for both public and private buckets. For private buckets, the recipient must use Bearer token authentication to access the resource.

## Default Headers

Every webhook request includes these headers:

| Header            | Value                        |
|-------------------|------------------------------|
| `Content-Type`    | `application/json`           |
| `User-Agent`      | `AOUI-Drive-Webhook/1.0`     |
| `X-Webhook-Event` | Event type (e.g., `resource.new`) |

Custom headers configured per webhook are added to these defaults.

## Request-Time Headers

In addition to configured webhook headers, you can pass optional headers at upload time that will be forwarded to webhook endpoints. This is useful for passing context-specific information like correlation IDs, authentication tokens, or custom metadata.

### How It Works

When uploading a resource, include headers with the `X-Webhook-Header-` prefix. These headers will be:
1. Extracted from the upload request
2. Stripped of the `X-Webhook-Header-` prefix
3. Included in the webhook HTTP request

### Header Precedence

Headers are applied in this order (later values override earlier ones):
1. Default headers (`Content-Type`, `User-Agent`, `X-Webhook-Event`)
2. Configured webhook headers (from database)
3. Request-time headers (from `X-Webhook-Header-*`)

### Examples

#### Stream Upload with Custom Headers

```bash
curl -X PUT "https://api.example.com/resources/{bucket_id}" \
  -H "Authorization: Bearer {your_token}" \
  -H "Content-Type: image/png" \
  -H "X-File-Extension: .png" \
  -H "X-Webhook-Header-Authorization: Bearer external-service-token" \
  -H "X-Webhook-Header-X-Correlation-Id: req-12345" \
  -H "X-Webhook-Header-X-Source-System: my-app" \
  --data-binary @image.png
```

The webhook request will include:
```
Authorization: Bearer external-service-token
X-Correlation-Id: req-12345
X-Source-System: my-app
```

#### Multipart Upload with Custom Headers

```bash
curl -X POST "https://api.example.com/resources/{bucket_id}" \
  -H "Authorization: Bearer {your_token}" \
  -H "X-Webhook-Header-X-Request-Id: abc-123" \
  -F "file=@document.pdf"
```

### Use Cases

| Use Case | Header Example |
|----------|----------------|
| Pass auth to external service | `X-Webhook-Header-Authorization: Bearer token` |
| Request tracing/correlation | `X-Webhook-Header-X-Correlation-Id: trace-123` |
| Source identification | `X-Webhook-Header-X-Source: mobile-app` |
| Custom metadata | `X-Webhook-Header-X-User-Id: user-456` |

## REST API Endpoints

All endpoints require Bearer token authentication.

### Webhook URL Management

| Method | Endpoint                                      | Description           |
|--------|-----------------------------------------------|-----------------------|
| POST   | `/buckets/:bucketId/webhooks`                 | Create webhook        |
| GET    | `/buckets/:bucketId/webhooks`                 | List webhooks         |
| GET    | `/buckets/:bucketId/webhooks/:id`             | Get webhook           |
| PUT    | `/buckets/:bucketId/webhooks/:id`             | Update webhook        |
| DELETE | `/buckets/:bucketId/webhooks/:id`             | Delete webhook        |

### Header Management

| Method | Endpoint                                              | Description     |
|--------|-------------------------------------------------------|-----------------|
| POST   | `/buckets/:bucketId/webhooks/:id/headers`             | Add header      |
| PUT    | `/buckets/:bucketId/webhooks/:id/headers/:headerId`   | Update header   |
| DELETE | `/buckets/:bucketId/webhooks/:id/headers/:headerId`   | Delete header   |

### Request/Response Examples

#### Create Webhook

```bash
POST /buckets/550e8400.../webhooks
Authorization: Bearer <token>
Content-Type: application/json

{
  "url": "https://api.example.com/webhooks/storage",
  "event_type": "resource.new",
  "is_active": true,
  "headers": [
    {"name": "X-API-Key", "value": "secret123"}
  ]
}
```

Response:
```json
{
  "success": true,
  "data": {
    "id": "770e8400-e29b-41d4-a716-446655440002",
    "bucket_id": "550e8400-e29b-41d4-a716-446655440000",
    "url": "https://api.example.com/webhooks/storage",
    "event_type": "resource.new",
    "is_active": true,
    "headers": [
      {"id": "...", "name": "X-API-Key", "value": "secret123", "created_at": "..."}
    ],
    "created_at": "2025-12-23T10:00:00Z",
    "updated_at": "2025-12-23T10:00:00Z"
  }
}
```

## UI Management

Webhooks can be managed via the web interface at:

```
/ui/buckets/:id/webhooks
```

Features:
- Add/delete webhook URLs
- Configure event types (resource.new, resource.deleted)
- Enable/disable webhooks
- Manage custom headers per webhook

## Implementation Flow

### Resource Upload Flow

```
1. Client uploads file to /resources/:bucketId (with optional X-Webhook-Header-* headers)
2. Controller extracts X-Webhook-Header-* headers from request
3. ResourceService.UploadStream() processes upload
4. On success, calls WebhookLauncher.TriggerEvent("resource.new", ..., extraHeaders)
5. WebhookService finds all active webhooks for bucket + event type
6. For each webhook, spawns goroutine to send HTTP POST
7. WebhookSender fetches configured headers and merges with extra headers
8. HTTP request sent with all headers applied
```

### Resource Delete Flow

```
1. Client calls DELETE /resources/:bucketId/:hash
2. ResourceService.Delete() removes resource
3. On success, calls WebhookLauncher.TriggerEvent("resource.deleted", ...)
4. Same dispatch flow as upload
```

## Code Structure

```
internal/features/webhook/
├── webhook.go              # Feature initialization
├── dto/
│   └── dto.go              # Request/Response DTOs, payload structs
├── repository/
│   └── repository.go       # Database operations
├── service/
│   ├── service.go          # Business logic, TriggerEvent()
│   └── dispatcher.go       # WebhookSender, HTTP delivery
└── controller/
    └── controller.go       # REST API handlers
```

## Dependency Injection

The webhook launcher is injected into the resource service at startup:

```go
// main.go
webhookFeature := webhook.New(db, bucketFeature.Repository)
resourceFeature := resource.New(db, bucketRepo, storagePath, publicURL, webhookFeature.Service)
```

This avoids circular dependencies while enabling the resource service to trigger webhook events.

## Configuration Notes

- Webhooks are sent asynchronously (fire-and-forget)
- HTTP timeout: 10 seconds per request
- No automatic retries (simplicity over complexity)
- Webhooks only trigger for active (`is_active = 1`) webhook URLs
- Deleting a webhook URL cascades to delete its headers
