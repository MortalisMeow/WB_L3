CREATE TABLE IF NOT EXISTS urls (
    id BIGSERIAL PRIMARY KEY,

    original_url TEXT NOT NULL,

    short_code VARCHAR(10) UNIQUE NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);