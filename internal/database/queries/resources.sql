-- name: GetResourceByID :one
SELECT id, bucket_id, hash, size, content_type, created_at
FROM resources WHERE id = ?;

-- name: GetResourceByBucketAndHash :one
SELECT id, bucket_id, hash, size, content_type, created_at
FROM resources WHERE bucket_id = ? AND hash = ?;

-- name: ListResourcesByBucketID :many
SELECT id, bucket_id, hash, size, content_type, created_at
FROM resources WHERE bucket_id = ? ORDER BY created_at DESC;

-- name: CreateResource :one
INSERT INTO resources (id, bucket_id, hash, size, content_type)
VALUES (?, ?, ?, ?, ?)
RETURNING id, bucket_id, hash, size, content_type, created_at;

-- name: DeleteResource :execrows
DELETE FROM resources WHERE id = ?;

-- name: DeleteResourceByBucketAndHash :execrows
DELETE FROM resources WHERE bucket_id = ? AND hash = ?;

-- name: ResourceExistsByBucketAndHash :one
SELECT EXISTS(SELECT 1 FROM resources WHERE bucket_id = ? AND hash = ?) AS resource_exists;
