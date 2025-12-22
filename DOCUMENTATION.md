# AOUI Drive - Technical Documentation

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Database Schema](#database-schema)
3. [Authentication & Authorization](#authentication--authorization)
4. [Feature Modules](#feature-modules)
5. [File Storage System](#file-storage-system)
6. [API Reference](#api-reference)
7. [Security](#security)
8. [Deployment](#deployment)

---

## Architecture Overview

AOUI Drive follows a clean, layered architecture with feature-based modularity. Each feature is self-contained with its own controller, service, repository, and DTOs.

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
    ├── Redis Cache                   │
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
        ├── Resource                  │
        │   ├── Repository ◄──────────┘
        │   ├── Service
        │   └── Controller
        │
        └── Health
            ├── Service
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
│ created_at      │       └─────────────────┘       │ created_at      │       │
│ updated_at      │                                 └─────────────────┘       │
└─────────────────┘                                                           │
                                                                              │
                                            UNIQUE(bucket_id, hash) ──────────┘
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
     ▼
┌──────────┐                           ┌──────────┐
│  Client  │ ──────────────────────►   │   API    │
│          │   Protected Request       │          │
│          │                           │          │
│          │   ◄────────────────────── │          │
│          │   Response                │          │
└──────────┘                           └──────────┘
```

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

**Key Files:**
- `service/service.go` - Authentication logic, JWT handling
- `controller/controller.go` - HTTP endpoints
- `repository/repository.go` - Client data access
- `dto/dto.go` - Request/response structures

### Bucket Feature

**Location:** `internal/features/bucket/`

**Responsibilities:**
- Bucket CRUD operations
- Bucket naming validation
- Public/private access control
- Storage directory management
- Public symlink creation/removal

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

**Upload Methods:**

1. **Streaming Upload (PUT)**
   - Raw body content
   - Requires `X-File-Extension` header
   - Best for large files

2. **Multipart Upload (POST)**
   - Form-data with `file` field
   - Extension extracted from filename
   - Standard browser-compatible upload

### Health Feature

**Location:** `internal/features/health/`

**Responsibilities:**
- Liveness probe (`/health`)
- Readiness probe (`/ready`)
- Database connectivity check
- Redis connectivity check

---

## File Storage System

### Directory Structure

```
{STORAGE_PATH}/
├── {bucket-uuid-1}/
│   ├── {sha256-hash-1}.jpg
│   ├── {sha256-hash-2}.pdf
│   └── {sha256-hash-3}.png
├── {bucket-uuid-2}/
│   └── {sha256-hash-4}.txt
└── public/
    ├── {bucket-uuid-1} -> ../{bucket-uuid-1}  (symlink)
    └── {bucket-uuid-3} -> ../{bucket-uuid-3}  (symlink)
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
└───────────────────┘    └───────────────────────────────┘
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

When a bucket is marked as public:

1. Symlink created: `{STORAGE_PATH}/public/{bucket-id} -> ../{bucket-id}`
2. Files accessible via: `GET /public/{bucket-id}/{hash}{extension}`
3. Public URL included in resource responses: `{PUBLIC_URL}/public/{bucket-id}/{filename}`

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

### Admin Endpoints

#### POST /admin/clients

Create new client (Admin only).

**Request:**
```json
{
  "name": "my-service",
  "role": "USER"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "my-service",
    "access_key": "AK...",
    "secret_key": "...",
    "role": "USER"
  }
}
```

#### POST /admin/clients/:id/regenerate-secret

Regenerate client secret key (Admin only).

**Response:**
```json
{
  "success": true,
  "data": {
    "secret_key": "new-secret-key..."
  }
}
```

### Bucket Endpoints

#### POST /buckets

Create new bucket.

**Query Parameters:**
- `public` (boolean) - Make bucket publicly accessible

**Request:**
```json
{
  "name": "my-bucket"
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "my-bucket",
    "public": false,
    "created_at": "2024-01-01T00:00:00Z"
  }
}
```

#### GET /buckets

List all buckets for authenticated client.

#### GET /buckets/:id

Get bucket details by ID.

#### DELETE /buckets/:id

Delete bucket by ID.

### Resource Endpoints

#### PUT /resources/:bucket

Upload resource via streaming.

**Headers:**
- `Content-Type` - MIME type of the file
- `X-File-Extension` - File extension (required, e.g., ".jpg")

**Body:** Raw file content

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "hash": "sha256...",
    "size": 1024,
    "content_type": "image/jpeg",
    "created_at": "2024-01-01T00:00:00Z",
    "public_url": "http://example.com/public/bucket-id/hash.jpg"
  }
}
```

#### POST /resources/:bucket

Upload resource via multipart form.

**Form Data:**
- `file` - File to upload

#### GET /resources/:bucket/:hash

Download resource by hash.

**Response Headers:**
- `X-Resource-Hash` - SHA-256 hash
- `Content-Type` - MIME type
- `Content-Length` - File size

#### HEAD /resources/:bucket/:hash

Get resource metadata without downloading.

#### GET /resources/:bucket

List all resources in bucket.

#### DELETE /resources/:bucket/:hash

Delete resource by hash.

### Health Endpoints

#### GET /health

Simple health check.

**Response:**
```json
{
  "status": "ok"
}
```

#### GET /ready

Readiness check with service status.

**Response:**
```json
{
  "status": "healthy",
  "services": {
    "database": "healthy",
    "cache": "healthy"
  }
}
```

---

## Security

### Authentication Security

- JWT tokens signed with HS256 algorithm
- Tokens expire after 24 hours
- Secret keys stored as bcrypt hashes (cost factor 10)
- Access keys are unique and indexed

### Data Isolation

- Clients can only access their own buckets
- Bucket ownership verified on every request
- Resource access requires bucket ownership

### Database Security

- Foreign key constraints enforced
- WAL mode for crash recovery
- Single connection limit prevents concurrent write conflicts

### Best Practices

1. **Change JWT Secret:** Set a strong `JWT_SECRET` in production
2. **Use HTTPS:** Deploy behind a reverse proxy with TLS
3. **Limit Access:** Use firewall rules to restrict database/Redis access
4. **Backup:** Regular backups of SQLite database and storage directory

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

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=secure-password
REDIS_DB=0

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

### Scaling Considerations

- **SQLite Limitations:** Single-writer, suitable for single-instance deployments
- **Storage:** Local filesystem; consider object storage for distributed deployments
- **Redis:** External Redis supports multiple instances
- **Stateless API:** Can run multiple instances with shared database/storage

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

### MIME Type to Extension

The system uses Go's `mime` package to determine file extensions from content types. For stream uploads, the `X-File-Extension` header is required to ensure correct file naming.
