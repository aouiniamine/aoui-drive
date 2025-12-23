# AOUI Drive

A file and media hosting server with a web dashboard for bucket and resource management, RESTful API, and webhook notifications for real-time event streaming.

## Features

- **File & Media Hosting** - Upload, store, and serve files with content-type detection
- **Public & Private Buckets** - Control access with public URLs or authenticated endpoints
- **Web Dashboard** - Browser-based UI for managing buckets, resources, and webhooks
- **Webhook Notifications** - Real-time HTTP callbacks on resource creation and deletion
- **Multi-tenant Architecture** - Clients are isolated with their own buckets and resources
- **JWT Authentication** - Secure API access with 24-hour tokens
- **Content Deduplication** - SHA-256 hash-based deduplication within buckets
- **Streaming Uploads** - Support for both streaming (PUT) and multipart form (POST) uploads
- **Swagger Documentation** - Built-in API documentation at `/swagger/`

## Tech Stack

- **Framework:** Echo v4
- **Database:** SQLite with WAL mode
- **Frontend:** HTMX + Tailwind CSS
- **Authentication:** JWT (HS256)
- **Password Hashing:** bcrypt

## Quick Start

### Prerequisites

- Go 1.21+
- SQLite3

### Installation

1. Clone the repository:
```bash
git clone https://github.com/aouiniamine/aoui-drive.git
cd aoui-drive
```

2. Copy the environment file:
```bash
cp .env.example .env
```

3. Configure your environment variables in `.env`

4. Run the application:
```bash
go run cmd/aoui-drive/main.go
```

### Create Your First Client

Use the CLI tool to create an admin client:
```bash
go run cmd/create-client/main.go -name "admin" -role "ADMIN"
```

This will output the `access_key` and `secret_key` for authentication.

## Web Dashboard

Access the web interface at:
```
http://localhost:8080/ui
```

Features:
- Login with access key and secret key
- Browse and manage buckets
- Upload, view, and delete resources
- Configure webhooks with custom headers
- Real-time updates with HTMX

## API Overview

### Authentication

```bash
# Login to get JWT token
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"access_key": "AK...", "secret_key": "..."}'
```

### Buckets

```bash
# Create a bucket
curl -X POST http://localhost:8080/buckets \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-bucket"}'

# Create a public bucket
curl -X POST "http://localhost:8080/buckets?public=true" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "public-bucket"}'

# List buckets
curl http://localhost:8080/buckets \
  -H "Authorization: Bearer <token>"
```

### Resources

```bash
# Upload via stream (PUT)
curl -X PUT http://localhost:8080/resources/<bucket-id> \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: image/jpeg" \
  -H "X-File-Extension: .jpg" \
  --data-binary @photo.jpg

# Upload via multipart form (POST)
curl -X POST http://localhost:8080/resources/<bucket-id> \
  -H "Authorization: Bearer <token>" \
  -F "file=@photo.jpg"

# Download resource
curl http://localhost:8080/resources/<bucket-id>/<hash> \
  -H "Authorization: Bearer <token>" \
  -o downloaded-file.jpg

# Access public resource (no auth required)
curl http://localhost:8080/public/<bucket-id>/<hash>.jpg \
  -o downloaded-file.jpg
```

### Webhooks

Configure webhooks to receive notifications when resources are created or deleted:

```bash
# Create a webhook
curl -X POST http://localhost:8080/buckets/<bucket-id>/webhooks \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://api.example.com/webhook",
    "event_type": "resource.new",
    "is_active": true,
    "headers": [{"name": "X-API-Key", "value": "secret"}]
  }'

# List webhooks
curl http://localhost:8080/buckets/<bucket-id>/webhooks \
  -H "Authorization: Bearer <token>"
```

Webhook payload example:
```json
{
  "event": "resource.new",
  "timestamp": "2025-12-23T10:30:00Z",
  "bucket_id": "...",
  "bucket_name": "my-bucket",
  "resource_id": "...",
  "resource_url": "https://example.com/resources/.../hash.jpg",
  "resource": {
    "hash": "abc123...",
    "size": 12345,
    "content_type": "image/jpeg",
    "extension": ".jpg"
  }
}
```

### Health Checks

```bash
# Simple health check
curl http://localhost:8080/health

# Readiness check (includes database status)
curl http://localhost:8080/ready
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST` | `0.0.0.0` | Server bind address |
| `PORT` | `8080` | Server port |
| `DATABASE_PATH` | `./data/aoui-drive.db` | SQLite database location |
| `STORAGE_PATH` | `./data/storage` | File storage directory |
| `PUBLIC_URL` | `` | Public URL prefix for resources |
| `JWT_SECRET` | `change-me-in-production` | JWT signing secret |
| `ENV` | `development` | Environment mode |

## Project Structure

```
aoui-drive/
├── cmd/
│   ├── aoui-drive/          # Main application
│   └── create-client/       # CLI for creating clients
├── internal/
│   ├── config/              # Configuration
│   ├── database/            # SQLite & migrations
│   ├── features/            # Feature modules
│   │   ├── auth/            # Authentication
│   │   ├── bucket/          # Bucket management
│   │   ├── health/          # Health checks
│   │   ├── resource/        # File management
│   │   ├── ui/              # Web dashboard
│   │   └── webhook/         # Webhook notifications
│   ├── middleware/          # Auth middleware
│   └── server/              # Echo server setup
├── pkg/
│   └── response/            # Response helpers
├── docs/                    # Swagger documentation
└── documetation/            # Architecture docs
```

## Documentation

- [Webhook Integration](documetation/webhooks.md) - Webhook architecture and configuration

## API Documentation (Swagger)

Access the interactive API documentation at:
```
http://localhost:8080/swagger/index.html
```

## Development

### Live Reload

Install Air for live reload during development:
```bash
go install github.com/cosmtrek/air@latest
air
```

### Generate SQLC Code

```bash
go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate
```

### Generate Swagger Docs

```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/aoui-drive/main.go
```

## License

Apache 2.0
