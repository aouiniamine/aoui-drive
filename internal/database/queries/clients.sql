-- name: GetClientByID :one
SELECT id, name, access_key, secret_key, role, is_active, created_at, updated_at
FROM clients WHERE id = ?;

-- name: GetClientByAccessKey :one
SELECT id, name, access_key, secret_key, role, is_active, created_at, updated_at
FROM clients WHERE access_key = ?;

-- name: ListClients :many
SELECT id, name, access_key, role, is_active, created_at, updated_at
FROM clients ORDER BY created_at DESC;

-- name: CreateClient :one
INSERT INTO clients (id, name, access_key, secret_key, role)
VALUES (?, ?, ?, ?, ?)
RETURNING id, name, access_key, secret_key, role, is_active, created_at, updated_at;

-- name: UpdateClient :one
UPDATE clients
SET name = ?, role = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
RETURNING id, name, access_key, secret_key, role, is_active, created_at, updated_at;

-- name: UpdateClientSecret :execrows
UPDATE clients SET secret_key = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?;

-- name: DeleteClient :exec
DELETE FROM clients WHERE id = ?;

-- name: ClientExistsByAccessKey :one
SELECT EXISTS(SELECT 1 FROM clients WHERE access_key = ?) AS client_exists;
