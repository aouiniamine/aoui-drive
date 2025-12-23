# AOUI Drive - Technical Documentation

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Database Schema](#database-schema)
3. [Authentication & Authorization](#authentication--authorization)
4. [Feature Modules](#feature-modules)
5. [File Storage System](#file-storage-system)
6. [Webhook System](#webhook-system)
7. [Web Dashboard](#web-dashboard)
8. [API Reference](#api-reference)
9. [Security](#security)
10. [Deployment](#deployment)

---

## Architecture Overview

AOUI Drive is a file and media hosting server with a clean, layered architecture and feature-based modularity. Each feature is self-contained with its own controller, service, repository, and DTOs.

### Layered Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    HTTP Layer (Echo)                     │
│                   Middleware (Auth, CORS)                │
├─────────────────────────────────────────────────────────┤
│                   Controller Layer                       │
│              Request/Response Handling                   │
├─────────────────────────────────────────────────────────┤
│                    Service Layer                         │
│               Business Logic & Validation                │
├─────────────────────────────────────────────────────────┤
│                   Repository Layer                       │
│                  Data Access Abstraction                 │
├─────────────────────────────────────────────────────────┤
│              Database (SQLite) / Storage                 │
└─────────────────────────────────────────────────────────┘
```

### Dependency Flow

```
main.go
    │
    ├── Database ─────────────────────┐
    │                                 │
    └── Features                      │
        ├── Auth                      │
        │   ├── Repository ◄──────────┤
        │   ├── Service               │
        │   └── Controller            │
        │                             │
        ├── Bucket                    │
        │   ├── Repository ◄──────────┤
        │   ├── Service               │
        │   └── Controller            │
        │                             │
        ├── Webhook ◄─────────────────┤
        │   ├── Repository            │
        │   ├── Service               │
        │   └── Controller            │
        │                             │
        ├── Resource                  │
        │   ├── Repository ◄──────────┘
        │   ├── Service ◄── WebhookLauncher
        │   └── Controller
        │
        ├── UI
        │   └── Controller (uses Auth, Bucket, Resource, Webhook)
        │
        └── Health
            └── Controller
```

### Response Format

All API responses follow a standardized envelope:

```json
{
  "success": true,
  "data": { },
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  },
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 100
  }
}
```

---

## Database Schema

AOUI Drive uses SQLite with WAL (Write-Ahead Logging) mode for improved concurrency and crash recovery.

### Entity Relationship Diagram

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│     clients     │       │     buckets     │       │    resources    │
├─────────────────┤       ├─────────────────┤       ├─────────────────┤
│ id (PK)         │       │ id (PK)         │       │ id (PK)         │
│ name            │◄──────│ client_id (FK)  │       │ bucket_id (FK)  │──────►│
│ access_key (UK) │       │ name            │◄──────│ hash            │       │
│ secret_key      │       │ is_public       │       │ size            │       │
│ role            │       │ created_at      │       │ content_type    │       │
│ is_active       │       │ updated_at      │       │ extension       │       │
│ created_at      │       └────────┬────────┘       │ created_at      │       │
│ updated_at      │                │                └─────────────────┘       │
└─────────────────┘                │                                          │
                                   │         UNIQUE(bucket_id, hash) ─────────┘
                                   │
                                   ▼
                    ┌──────────────────────────┐
                    │      webhook_urls        │
                    ├──────────────────────────┤
                    │ id (PK)                  │
                    │ bucket_id (FK)           │
                    │ url                      │
                    │ event_type               │
                    │ is_active                │
                    │ created_at               │
                    │ updated_at               │
                    └────────────┬─────────────┘
                                 │
                                 ▼
                    ┌──────────────────────────┐
                    │     webhook_headers      │
                    ├──────────────────────────┤
                    │ id (PK)                  │
                    │ webhook_url_id (FK)      │
                    │ header_name              │
                    │ header_value             │
                    │ created_at               │
                    └──────────────────────────┘
```

### Clients Table

Stores API clients (users) with authentication credentials.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `name` | TEXT | Client display name |
| `access_key` | TEXT | Unique public key (format: `AK{hex}`) |
| `secret_key` | TEXT | Bcrypt-hashed private key |
| `role` | TEXT | ADMIN, MANAGER, or USER |
| `is_active` | INTEGER | 1 = active, 0 = disabled |
| `created_at` | DATETIME | Creation timestamp |
| `updated_at` | DATETIME | Last update timestamp |

### Buckets Table

Storage containers owned by clients.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `name` | TEXT | Bucket name (3-63 chars, lowercase alphanumeric) |
| `client_id` | TEXT | Owner client reference |
| `is_public` | INTEGER | 1 = public access, 0 = private |
| `created_at` | DATETIME | Creation timestamp |
| `updated_at` | DATETIME | Last update timestamp |

**Constraints:**
- `UNIQUE(name, client_id)` - Bucket names unique per client
- `FOREIGN KEY (client_id) REFERENCES clients(id)`

### Resources Table

Files stored within buckets.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `bucket_id` | TEXT | Parent bucket reference |
| `hash` | TEXT | SHA-256 hash of file content |
| `size` | INTEGER | File size in bytes |
| `content_type` | TEXT | MIME type |
| `extension` | TEXT | File extension (e.g., ".jpg") |
| `created_at` | DATETIME | Creation timestamp |

**Constraints:**
- `UNIQUE(bucket_id, hash)` - Enables deduplication within bucket
- `FOREIGN KEY (bucket_id) REFERENCES buckets(id) ON DELETE CASCADE`

### Webhook URLs Table

Webhook endpoints configured per bucket.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `bucket_id` | TEXT | Parent bucket reference |
| `url` | TEXT | Webhook endpoint URL |
| `event_type` | TEXT | `resource.new` or `resource.deleted` |
| `is_active` | INTEGER | 1 = active, 0 = disabled |
| `created_at` | DATETIME | Creation timestamp |
| `updated_at` | DATETIME | Last update timestamp |

### Webhook Headers Table

Custom HTTP headers for webhook requests.

| Column | Type | Description |
|--------|------|-------------|
| `id` | TEXT | UUID primary key |
| `webhook_url_id` | TEXT | Parent webhook reference |
| `header_name` | TEXT | HTTP header name |
| `header_value` | TEXT | HTTP header value |
| `created_at` | DATETIME | Creation timestamp |

---

## Authentication & Authorization

### JWT Token Flow

```
┌──────────┐     POST /auth/login      ┌──────────┐
│  Client  │ ──────────────────────►   │   API    │
│          │   {access_key, secret}    │          │
│          │                           │          │
│          │   ◄────────────────────── │          │
│          │   {access_token, expires} │          │
└──────────┘                           └──────────┘
     │
     │  Authorization: Bearer <token>
     │  OR Cookie: session=<token>
     ▼
┌──────────┐                           ┌──────────┐
│  Client  │ ──────────────────────►   │   API    │
│          │   Protected Request       │          │
│          │                           │          │
│          │   ◄────────────────────── │          │
│          │   Response                │          │
└──────────┘                           └──────────┘
```

### Unified Authentication

The auth middleware supports both:
1. **Bearer Token** - `Authorization: Bearer <token>` header
2. **Session Cookie** - `session` cookie (used by web dashboard)

For API routes, authentication failures return JSON errors.
For UI routes (`/ui/*`), authentication failures redirect to login page.

### Token Structure

JWT tokens contain the following claims:

```json
{
  "client_id": "uuid",
  "role": "USER",
  "exp": 1703289600,
  "iat": 1703203200
}
```

- **Expiration:** 24 hours from issuance
- **Algorithm:** HS256

### Role-Based Access Control

| Role | Permissions |
|------|-------------|
| `ADMIN` | Full access, can manage clients |
| `MANAGER` | Manage own buckets and resources |
| `USER` | Read/write access to own resources |

### Credential Generation

- **Access Key:** 16 random bytes, hex-encoded, prefixed with "AK"
  - Example: `AK1a2b3c4d5e6f7890abcdef12345678`
- **Secret Key:** 32 random bytes, hex-encoded
  - Stored as bcrypt hash (cost factor 10)

---

## Feature Modules

### Auth Feature

**Location:** `internal/features/auth/`

**Responsibilities:**
- Client authentication (login)
- JWT token generation and validation
- Client management (admin only)
- Secret key regeneration

### Bucket Feature

**Location:** `internal/features/bucket/`

**Responsibilities:**
- Bucket CRUD operations
- Bucket naming validation
- Public/private access control
- Storage directory management

**Bucket Naming Rules:**
- Length: 3-63 characters
- Allowed: lowercase letters, numbers, hyphens, periods
- Must start and end with alphanumeric character
- Regex: `^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$`

### Resource Feature

**Location:** `internal/features/resource/`

**Responsibilities:**
- File upload (streaming and multipart)
- Content deduplication via SHA-256
- File download and metadata
- Resource listing and deletion
- Public URL generation
- Webhook event triggering

**Upload Methods:**

1. **Streaming Upload (PUT)**
   - Raw body content
   - Optional `X-File-Extension` header
   - Best for large files

2. **Multipart Upload (POST)**
   - Form-data with `file` field
   - Extension extracted from filename
   - Standard browser-compatible upload

### Webhook Feature

**Location:** `internal/features/webhook/`

**Responsibilities:**
- Webhook URL CRUD operations
- Custom header management
- Event dispatching on resource changes
- HTTP delivery to external endpoints

### UI Feature

**Location:** `internal/features/ui/`

**Responsibilities:**
- Web dashboard interface
- Session-based authentication (cookie)
- Bucket and resource management UI
- Webhook configuration UI
- File upload/download handling

### Health Feature

**Location:** `internal/features/health/`

**Responsibilities:**
- Liveness probe (`/health`)
- Readiness probe (`/ready`)
- Database connectivity check

---

## File Storage System

### Directory Structure

```
{STORAGE_PATH}/
├── {bucket-uuid-1}/
│   ├── {sha256-hash-1}.jpg
│   ├── {sha256-hash-2}.pdf
│   └── {sha256-hash-3}.png
└── {bucket-uuid-2}/
    └── {sha256-hash-4}.txt
```

### Upload Process

```
┌─────────────────────────────────────────────────────────┐
│                    Upload Request                        │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│              Verify Bucket Ownership                     │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│          Stream to Temp File + Compute SHA-256           │
└─────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────┐
│            Check for Existing Hash (Dedup)               │
└─────────────────────────────────────────────────────────┘
                    │              │
            Exists  │              │  New
                    ▼              ▼
┌───────────────────┐    ┌───────────────────────────────┐
│  Return Existing  │    │  Move to Final Location       │
│     Resource      │    │  Create Database Record       │
└───────────────────┘    │  Trigger Webhook Event        │
                         └───────────────────────────────┘
                                   │
                                   ▼
                    ┌───────────────────────────────────┐
                    │        Return Resource Info        │
                    │   (include public_url if public)   │
                    └───────────────────────────────────┘
```

### Deduplication

Resources are deduplicated within each bucket using SHA-256 hashes:

1. File content is hashed during upload
2. Hash is checked against existing resources in the bucket
3. If match found, existing resource is returned (no duplicate storage)
4. Hash becomes part of the filename: `{hash}{extension}`

### Public Access

Public bucket files are accessible via static file serving:
- URL: `GET /public/{bucket-id}/{hash}{extension}`
- No authentication required
- Files served directly from storage directory

---

## Webhook System

### Architecture

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Resource       │────▶│  Webhook         │────▶│  External       │
│  Service        │     │  Service         │     │  Endpoint       │
│                 │     │                  │     │                 │
│  - Upload       │     │  - TriggerEvent  │     │  HTTP POST      │
│  - Delete       │     │  - SendWebhook   │     │  + Headers      │
└─────────────────┘     └──────────────────┘     └─────────────────┘
```

### Event Types

| Event Type         | Trigger                          |
|--------------------|----------------------------------|
| `resource.new`     | When a new resource is uploaded  |
| `resource.deleted` | When a resource is deleted       |

### Webhook Payload

```json
{
  "event": "resource.new",
  "timestamp": "2025-12-23T10:30:00Z",
  "bucket_id": "550e8400-e29b-41d4-a716-446655440000",
  "bucket_name": "my-bucket",
  "resource_id": "660e8400-e29b-41d4-a716-446655440001",
  "resource_url": "https://example.com/resources/{bucket}/{hash}.{ext}",
  "resource": {
    "hash": "abc123def456...",
    "size": 12345,
    "content_type": "image/png",
    "extension": ".png"
  }
}
```

### Default Headers

Every webhook request includes:

| Header            | Value                        |
|-------------------|------------------------------|
| `Content-Type`    | `application/json`           |
| `User-Agent`      | `AOUI-Drive-Webhook/1.0`     |
| `X-Webhook-Event` | Event type                   |

Custom headers configured per webhook are added to these defaults.

### Delivery

- Webhooks are sent asynchronously (fire-and-forget)
- HTTP timeout: 10 seconds per request
- No automatic retries (simplicity over complexity)
- Only active webhooks (`is_active = 1`) receive events

---

## Web Dashboard

### Access

```
http://localhost:8080/ui
```

### Features

- **Login** - Authenticate with access key and secret key
- **Bucket Management** - View, create buckets
- **Resource Management** - Upload, view, download, delete files
- **Webhook Configuration** - Add webhooks, manage custom headers
- **Real-time Updates** - HTMX-powered dynamic interface

### Technology

- **Frontend:** HTMX + Tailwind CSS
- **Templating:** Go html/template
- **Session:** JWT stored in HTTP-only cookie

---

## API Reference

### Authentication Endpoints

#### POST /auth/login

Authenticate and receive JWT token.

**Request:**
```json
{
  "access_key": "AK1a2b3c4d5e6f7890",
  "secret_key": "abcdef123456..."
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_in": 86400
  }
}
```

### Bucket Endpoints

#### POST /buckets

Create new bucket.

#### GET /buckets

List all buckets for authenticated client.

#### GET /buckets/:id

Get bucket details by ID.

#### DELETE /buckets/:id

Delete bucket by ID.

### Resource Endpoints

#### PUT /resources/:bucket

Upload resource via streaming.

#### POST /resources/:bucket

Upload resource via multipart form.

#### GET /resources/:bucket/:hash

Download resource by hash.

#### HEAD /resources/:bucket/:hash

Get resource metadata without downloading.

#### GET /resources/:bucket

List all resources in bucket.

#### DELETE /resources/:bucket/:hash

Delete resource by hash.

### Webhook Endpoints

#### POST /buckets/:bucketId/webhooks

Create webhook URL.

#### GET /buckets/:bucketId/webhooks

List webhooks for bucket.

#### GET /buckets/:bucketId/webhooks/:id

Get webhook details.

#### PUT /buckets/:bucketId/webhooks/:id

Update webhook.

#### DELETE /buckets/:bucketId/webhooks/:id

Delete webhook.

#### POST /buckets/:bucketId/webhooks/:id/headers

Add custom header.

#### DELETE /buckets/:bucketId/webhooks/:id/headers/:headerId

Delete custom header.

### Health Endpoints

#### GET /health

Simple health check.

#### GET /ready

Readiness check with database status.

---

## Security

### Authentication Security

- JWT tokens signed with HS256 algorithm
- Tokens expire after 24 hours
- Secret keys stored as bcrypt hashes (cost factor 10)
- Access keys are unique and indexed
- Session cookies are HTTP-only

### Data Isolation

- Clients can only access their own buckets
- Bucket ownership verified on every request
- Resource access requires bucket ownership

### Database Security

- Foreign key constraints enforced
- WAL mode for crash recovery
- Cascading deletes for referential integrity

### Best Practices

1. **Change JWT Secret:** Set a strong `JWT_SECRET` in production
2. **Use HTTPS:** Deploy behind a reverse proxy with TLS
3. **Backup:** Regular backups of SQLite database and storage directory

---

## Deployment

### Environment Variables

```bash
# Server
HOST=0.0.0.0
PORT=8080

# Database
DATABASE_PATH=/data/aoui-drive.db

# Storage
STORAGE_PATH=/data/storage
PUBLIC_URL=https://cdn.example.com

# Security
JWT_SECRET=your-secure-secret-key

# Environment
ENV=production
```

### Docker Deployment

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o aoui-drive ./cmd/aoui-drive

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/aoui-drive .
EXPOSE 8080
CMD ["./aoui-drive"]
```

### Health Checks

Configure your orchestrator to use:
- **Liveness:** `GET /health`
- **Readiness:** `GET /ready`

---

## Appendix

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Malformed request body |
| `UNAUTHORIZED` | 401 | Invalid or missing token |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource already exists |
| `INTERNAL_ERROR` | 500 | Server error |
