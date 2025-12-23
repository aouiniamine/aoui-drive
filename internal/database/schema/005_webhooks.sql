-- Webhook URLs table
CREATE TABLE IF NOT EXISTS webhook_urls (
    id TEXT PRIMARY KEY,
    bucket_id TEXT NOT NULL,
    url TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('resource.new', 'resource.deleted')),
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (bucket_id) REFERENCES buckets(id) ON DELETE CASCADE,
    UNIQUE(bucket_id, url, event_type)
);

CREATE INDEX IF NOT EXISTS idx_webhook_urls_bucket_id ON webhook_urls(bucket_id);
CREATE INDEX IF NOT EXISTS idx_webhook_urls_event_type ON webhook_urls(event_type);
CREATE INDEX IF NOT EXISTS idx_webhook_urls_is_active ON webhook_urls(is_active);

-- Webhook Headers table (many-to-one with webhook_urls)
CREATE TABLE IF NOT EXISTS webhook_headers (
    id TEXT PRIMARY KEY,
    webhook_url_id TEXT NOT NULL,
    header_name TEXT NOT NULL,
    header_value TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (webhook_url_id) REFERENCES webhook_urls(id) ON DELETE CASCADE,
    UNIQUE(webhook_url_id, header_name)
);

CREATE INDEX IF NOT EXISTS idx_webhook_headers_webhook_url_id ON webhook_headers(webhook_url_id);

-- Webhook Events table (for tracking and retry logic)
CREATE TABLE IF NOT EXISTS webhook_events (
    id TEXT PRIMARY KEY,
    webhook_url_id TEXT NOT NULL,
    bucket_id TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('resource.new', 'resource.deleted')),
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'success', 'failed', 'retrying')),
    payload TEXT NOT NULL,
    response_code INTEGER,
    response_body TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    next_retry_at DATETIME,
    last_attempt_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (webhook_url_id) REFERENCES webhook_urls(id) ON DELETE CASCADE,
    FOREIGN KEY (bucket_id) REFERENCES buckets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_webhook_url_id ON webhook_events(webhook_url_id);
CREATE INDEX IF NOT EXISTS idx_webhook_events_bucket_id ON webhook_events(bucket_id);
CREATE INDEX IF NOT EXISTS idx_webhook_events_status ON webhook_events(status);
CREATE INDEX IF NOT EXISTS idx_webhook_events_next_retry_at ON webhook_events(next_retry_at);
