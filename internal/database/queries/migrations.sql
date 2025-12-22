-- name: GetAppliedMigrations :many
SELECT version, applied_at FROM schema_migrations ORDER BY version;

-- name: InsertMigration :exec
INSERT INTO schema_migrations (version) VALUES (?);
