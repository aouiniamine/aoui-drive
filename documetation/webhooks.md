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
1. Client uploads file to /resources/:bucketId
2. ResourceService.UploadStream() processes upload
3. On success, calls WebhookLauncher.TriggerEvent("resource.new", ...)
4. WebhookService finds all active webhooks for bucket + event type
5. For each webhook, spawns goroutine to send HTTP POST
6. WebhookSender fetches headers and sends request
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
