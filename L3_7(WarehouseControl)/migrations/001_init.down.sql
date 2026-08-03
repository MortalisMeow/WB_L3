DROP TRIGGER IF EXISTS items_audit_trigger ON items;
DROP FUNCTION IF EXISTS log_item_change();
DROP TABLE IF EXISTS item_history;
DROP TABLE IF EXISTS items;
