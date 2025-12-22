-- name: GetBucketByID :one
SELECT id, name, client_id, is_public, created_at, updated_at
FROM buckets WHERE id = ?;

-- name: GetBucketByNameAndClientID :one
SELECT id, name, client_id, is_public, created_at, updated_at
FROM buckets WHERE name = ? AND client_id = ?;

-- name: ListBuckets :many
SELECT id, name, client_id, is_public, created_at, updated_at
FROM buckets ORDER BY name;

-- name: ListBucketsByClientID :many
SELECT id, name, client_id, is_public, created_at, updated_at
FROM buckets WHERE client_id = ? ORDER BY name;

-- name: CreateBucket :one
INSERT INTO buckets (id, name, client_id, is_public)
VALUES (?, ?, ?, ?)
RETURNING id, name, client_id, is_public, created_at, updated_at;

-- name: DeleteBucket :execrows
DELETE FROM buckets WHERE id = ?;

-- name: BucketExistsByNameAndClientID :one
SELECT EXISTS(SELECT 1 FROM buckets WHERE name = ? AND client_id = ?) AS bucket_exists;

-- name: GetPublicBucketByName :one
SELECT id, name, client_id, is_public, created_at, updated_at
FROM buckets WHERE name = ? AND is_public = 1;
