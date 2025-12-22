CREATE TABLE IF NOT EXISTS clients (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    access_key TEXT NOT NULL UNIQUE,
    secret_key TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('ADMIN', 'MANAGER', 'USER')) DEFAULT 'USER',
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_clients_access_key ON clients(access_key);
