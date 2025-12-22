CREATE TABLE IF NOT EXISTS resources (
    id TEXT PRIMARY KEY,
    bucket_id TEXT NOT NULL,
    hash TEXT NOT NULL,
    size INTEGER NOT NULL,
    content_type TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bucket_id) REFERENCES buckets(id) ON DELETE CASCADE,
    UNIQUE(bucket_id, hash)
);

CREATE INDEX IF NOT EXISTS idx_resources_bucket_id ON resources(bucket_id);
CREATE INDEX IF NOT EXISTS idx_resources_hash ON resources(hash);
