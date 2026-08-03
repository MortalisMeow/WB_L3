CREATE TABLE IF NOT EXISTS items (
    id          BIGSERIAL PRIMARY KEY,
    type        VARCHAR(10) NOT NULL CHECK (type IN ('income', 'expense')),
    amount      NUMERIC(12, 2) NOT NULL CHECK (amount >= 0),
    category    VARCHAR(100) NOT NULL DEFAULT '',
    occurred_at DATE NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_items_occurred_at ON items (occurred_at);
CREATE INDEX IF NOT EXISTS idx_items_type ON items (type);
CREATE INDEX IF NOT EXISTS idx_items_category ON items (category);
