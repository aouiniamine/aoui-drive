# AOUI Drive

A MinIO-like Object Storage API built in Go. AOUI Drive provides a secure, multi-tenant storage service with authentication, bucket management, and resource (file) handling capabilities.

## Features

- **Multi-tenant Architecture** - Clients are isolated with their own buckets and resources
- **JWT Authentication** - Secure API access with 24-hour tokens
- **Role-Based Access Control** - ADMIN, MANAGER, and USER roles
- **Content Deduplication** - SHA-256 hash-based deduplication within buckets
- **Public Buckets** - Optional public access via symlinks
- **Streaming Uploads** - Support for both streaming (PUT) and multipart form (POST) uploads
- **Swagger Documentation** - Built-in API documentation at `/swagger/`

## Tech Stack

- **Framework:** Echo v4
- **Database:** SQLite with WAL mode
- **Cache:** Redis
- **Authentication:** JWT (HS256)
- **Password Hashing:** bcrypt

## Quick Start

### Prerequisites

- Go 1.21+
- Redis server
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

4. Start Redis (using Docker):
```bash
docker-compose up -d
```

5. Run the application:
```bash
go run cmd/aoui-drive/main.go
```

### Create Your First Client

Use the CLI tool to create an admin client:
```bash
go run cmd/create-client/main.go -name "admin" -role "ADMIN"
```

This will output the `access_key` and `secret_key` for authentication.

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

# List resources in bucket
curl http://localhost:8080/resources/<bucket-id> \
  -H "Authorization: Bearer <token>"
```

### Health Checks

```bash
# Simple health check
curl http://localhost:8080/health

# Readiness check (includes database and Redis status)
curl http://localhost:8080/ready
```

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HOST` | `0.0.0.0` | Server bind address |
| `PORT` | `8080` | Server port |
| `DATABASE_PATH` | `./data/aoui-drive.db` | SQLite database location |
| `STORAGE_PATH` | `./data/storage` | File storage directory |
| `PUBLIC_URL` | `` | Public URL prefix for public resources |
| `REDIS_HOST` | `localhost` | Redis server hostname |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | `` | Redis password |
| `REDIS_DB` | `0` | Redis database number |
| `JWT_SECRET` | `change-me-in-production` | JWT signing secret |
| `ENV` | `development` | Environment mode |

## Project Structure

```
aoui-drive/
├── cmd/
│   ├── aoui-drive/          # Main application
│   └── create-client/       # CLI for creating clients
├── internal/
│   ├── cache/               # Redis client
│   ├── config/              # Configuration
│   ├── database/            # SQLite & migrations
│   ├── features/            # Feature modules
│   │   ├── auth/            # Authentication
│   │   ├── bucket/          # Bucket management
│   │   ├── health/          # Health checks
│   │   └── resource/        # File management
│   ├── middleware/          # Auth middleware
│   └── server/              # Echo server setup
├── pkg/
│   └── response/            # Response helpers
└── docs/                    # Swagger documentation
```

## Documentation

For detailed architecture and API documentation, see [DOCUMENTATION.md](DOCUMENTATION.md).

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
