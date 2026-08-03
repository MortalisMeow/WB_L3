CREATE TABLE IF NOT EXISTS items (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    sku         VARCHAR(100) NOT NULL UNIQUE,
    quantity    INTEGER NOT NULL DEFAULT 0 CHECK (quantity >= 0),
    price       NUMERIC(12, 2) NOT NULL DEFAULT 0 CHECK (price >= 0),
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_items_sku ON items (sku);

CREATE TABLE IF NOT EXISTS item_history (
    id          BIGSERIAL PRIMARY KEY,
    item_id     BIGINT NOT NULL,
    action      VARCHAR(10) NOT NULL CHECK (action IN ('INSERT', 'UPDATE', 'DELETE')),
    changed_by  VARCHAR(100) NOT NULL DEFAULT 'system',
    changed_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    old_data    JSONB,
    new_data    JSONB
);

CREATE INDEX IF NOT EXISTS idx_item_history_item_id ON item_history (item_id);
CREATE INDEX IF NOT EXISTS idx_item_history_changed_at ON item_history (changed_at);
CREATE INDEX IF NOT EXISTS idx_item_history_changed_by ON item_history (changed_by);
CREATE INDEX IF NOT EXISTS idx_item_history_action ON item_history (action);

CREATE OR REPLACE FUNCTION log_item_change() RETURNS TRIGGER AS $$
DECLARE
    v_user TEXT;
BEGIN
    v_user := COALESCE(NULLIF(current_setting('app.current_user', true), ''), 'system');

    IF TG_OP = 'INSERT' THEN
        INSERT INTO item_history (item_id, action, changed_by, old_data, new_data)
        VALUES (NEW.id, 'INSERT', v_user, NULL, to_jsonb(NEW));
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO item_history (item_id, action, changed_by, old_data, new_data)
        VALUES (NEW.id, 'UPDATE', v_user, to_jsonb(OLD), to_jsonb(NEW));
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO item_history (item_id, action, changed_by, old_data, new_data)
        VALUES (OLD.id, 'DELETE', v_user, to_jsonb(OLD), NULL);
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS items_audit_trigger ON items;
CREATE TRIGGER items_audit_trigger
AFTER INSERT OR UPDATE OR DELETE ON items
FOR EACH ROW EXECUTE FUNCTION log_item_change();
