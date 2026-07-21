CREATE TABLE IF NOT EXISTS clicks (
    id BIGSERIAL PRIMARY KEY,

    url_id BIGINT NOT NULL,

    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    user_agent TEXT,

    ip_address VARCHAR(45),

    referer TEXT,

    CONSTRAINT fk_clicks_url_id FOREIGN KEY (url_id) REFERENCES urls(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_clicks_url_id ON clicks(url_id);

CREATE INDEX IF NOT EXISTS idx_clicks_url_id_clicked_at ON clicks(url_id, clicked_at);