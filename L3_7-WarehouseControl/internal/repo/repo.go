package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"warehousecontrol/internal/models"

	"github.com/lib/pq"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Migrate() error {
	_, err := r.db.Exec(`
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
	`)
	return err
}

func (r *Repo) withUser(ctx context.Context, username string, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `SELECT set_config('app.current_user', $1, true)`, username); err != nil {
		return fmt.Errorf("set user: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repo) Create(ctx context.Context, username string, item *models.Item) (*models.Item, error) {
	err := r.withUser(ctx, username, func(tx *sql.Tx) error {
		query := `
			INSERT INTO items (name, sku, quantity, price, description)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id, created_at, updated_at
		`
		return tx.QueryRowContext(ctx, query, item.Name, item.SKU, item.Quantity, item.Price, item.Description).
			Scan(&item.ID, &item.CreatedAt, &item.UpdatedAt)
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, models.ErrDuplicateSKU
		}
		return nil, fmt.Errorf("insert item: %w", err)
	}
	return item, nil
}

func (r *Repo) GetByID(ctx context.Context, id int64) (*models.Item, error) {
	item := &models.Item{}
	query := `
		SELECT id, name, sku, quantity, price, description, created_at, updated_at
		FROM items WHERE id = $1
	`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&item.ID, &item.Name, &item.SKU, &item.Quantity, &item.Price, &item.Description, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("get item: %w", err)
	}
	return item, nil
}

func (r *Repo) List(ctx context.Context) ([]models.Item, error) {
	query := `
		SELECT id, name, sku, quantity, price, description, created_at, updated_at
		FROM items
		ORDER BY id DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list items: %w", err)
	}
	defer rows.Close()

	var list []models.Item
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(&item.ID, &item.Name, &item.SKU, &item.Quantity, &item.Price, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	if list == nil {
		list = []models.Item{}
	}
	return list, rows.Err()
}

func (r *Repo) Update(ctx context.Context, username string, item *models.Item) error {
	return r.withUser(ctx, username, func(tx *sql.Tx) error {
		query := `
			UPDATE items
			SET name = $1, sku = $2, quantity = $3, price = $4, description = $5, updated_at = NOW()
			WHERE id = $6
		`
		res, err := tx.ExecContext(ctx, query, item.Name, item.SKU, item.Quantity, item.Price, item.Description, item.ID)
		if err != nil {
			if isUniqueViolation(err) {
				return models.ErrDuplicateSKU
			}
			return fmt.Errorf("update item: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return models.ErrNotFound
		}
		return nil
	})
}

func (r *Repo) Delete(ctx context.Context, username string, id int64) error {
	return r.withUser(ctx, username, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx, `DELETE FROM items WHERE id = $1`, id)
		if err != nil {
			return fmt.Errorf("delete item: %w", err)
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			return models.ErrNotFound
		}
		return nil
	})
}

func (r *Repo) ListHistory(ctx context.Context, f models.HistoryFilter) ([]models.HistoryEntry, error) {
	query := `
		SELECT id, item_id, action, changed_by, changed_at, old_data, new_data
		FROM item_history
		WHERE ($1 = 0 OR item_id = $1)
		  AND changed_at >= $2::timestamptz
		  AND changed_at <= $3::timestamptz
		  AND ($4 = '' OR changed_by = $4)
		  AND ($5 = '' OR action = $5)
		ORDER BY changed_at DESC, id DESC
	`
	from := defaultTS(f.From, "1970-01-01 00:00:00+00")
	to := defaultTS(f.To, "2099-12-31 23:59:59+00")

	rows, err := r.db.QueryContext(ctx, query, f.ItemID, from, to, f.User, f.Action)
	if err != nil {
		return nil, fmt.Errorf("list history: %w", err)
	}
	defer rows.Close()

	var list []models.HistoryEntry
	for rows.Next() {
		var e models.HistoryEntry
		var oldData, newData sql.NullString
		if err := rows.Scan(&e.ID, &e.ItemID, &e.Action, &e.ChangedBy, &e.ChangedAt, &oldData, &newData); err != nil {
			return nil, err
		}
		if oldData.Valid {
			e.OldData = json.RawMessage(oldData.String)
		}
		if newData.Valid {
			e.NewData = json.RawMessage(newData.String)
		}
		list = append(list, e)
	}
	if list == nil {
		list = []models.HistoryEntry{}
	}
	return list, rows.Err()
}

func (r *Repo) GetHistoryByID(ctx context.Context, id int64) (*models.HistoryEntry, error) {
	e := &models.HistoryEntry{}
	var oldData, newData sql.NullString
	query := `
		SELECT id, item_id, action, changed_by, changed_at, old_data, new_data
		FROM item_history WHERE id = $1
	`
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&e.ID, &e.ItemID, &e.Action, &e.ChangedBy, &e.ChangedAt, &oldData, &newData)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("get history: %w", err)
	}
	if oldData.Valid {
		e.OldData = json.RawMessage(oldData.String)
	}
	if newData.Valid {
		e.NewData = json.RawMessage(newData.String)
	}
	return e, nil
}

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return pqErr.Code == "23505"
	}
	return false
}

func defaultTS(s, def string) string {
	if strings.TrimSpace(s) == "" {
		return def
	}
	if len(s) == 10 {
		return s + " 00:00:00+00"
	}
	return s
}
