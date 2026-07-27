package repository

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"commenttree/internal/models"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(body, author string, parentID *int64) (*models.Comment, error) {
	var path string
	if parentID != nil {
		err := r.db.QueryRow("SELECT path FROM comments WHERE id = ?", *parentID).Scan(&path)
		if err != nil {
			return nil, fmt.Errorf("parent not found: %w", err)
		}
	}

	res, err := r.db.Exec("INSERT INTO comments (parent_id, path, body, author) VALUES (?, ?, ?, ?)",
		parentID, "", body, author)
	if err != nil {
		return nil, err
	}

	id, _ := res.LastInsertId()

	if parentID != nil {
		path = fmt.Sprintf("%s%d/", path, id)
	} else {
		path = fmt.Sprintf("/%d/", id)
	}

	_, err = r.db.Exec("UPDATE comments SET path = ? WHERE id = ?", path, id)
	if err != nil {
		return nil, err
	}

	return r.GetByID(id)
}

func (r *Repo) GetByID(id int64) (*models.Comment, error) {
	c := &models.Comment{}
	var parentID sql.NullInt64
	err := r.db.QueryRow("SELECT id, parent_id, path, body, author, created_at FROM comments WHERE id = ?", id).
		Scan(&c.ID, &parentID, &c.Path, &c.Body, &c.Author, &c.CreatedAt)
	if err != nil {
		return nil, err
	}
	if parentID.Valid {
		c.ParentID = &parentID.Int64
	}
	return c, nil
}

func (r *Repo) GetTree(rootID int64) ([]*models.Comment, error) {
	var rootPath string
	err := r.db.QueryRow("SELECT path FROM comments WHERE id = ?", rootID).Scan(&rootPath)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query("SELECT id, parent_id, path, body, author, created_at FROM comments WHERE path LIKE ? || '%' ORDER BY path",
		rootPath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &parentID, &c.Path, &c.Body, &c.Author, &c.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			c.ParentID = &parentID.Int64
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *Repo) GetRoots(limit, offset int) ([]*models.Comment, error) {
	rows, err := r.db.Query("SELECT id, parent_id, path, body, author, created_at FROM comments WHERE parent_id IS NULL ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &parentID, &c.Path, &c.Body, &c.Author, &c.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			c.ParentID = &parentID.Int64
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *Repo) CountRoots() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM comments WHERE parent_id IS NULL").Scan(&count)
	return count, err
}

func (r *Repo) DeleteTree(rootID int64) error {
	var rootPath string
	err := r.db.QueryRow("SELECT path FROM comments WHERE id = ?", rootID).Scan(&rootPath)
	if err != nil {
		return err
	}
	_, err = r.db.Exec("DELETE FROM comments WHERE path LIKE ? || '%'", rootPath)
	return err
}

func (r *Repo) Search(query string) ([]*models.Comment, error) {
	rows, err := r.db.Query(`
		SELECT c.id, c.parent_id, c.path, c.body, c.author, c.created_at
		FROM comments_fts fts
		JOIN comments c ON c.rowid = fts.rowid
		WHERE comments_fts MATCH ?
		ORDER BY rank
		LIMIT 50
	`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		var parentID sql.NullInt64
		if err := rows.Scan(&c.ID, &parentID, &c.Path, &c.Body, &c.Author, &c.CreatedAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			c.ParentID = &parentID.Int64
		}
		c.Matched = true
		list = append(list, c)
	}
	return list, nil
}

func (r *Repo) GetPaths(ids []int64) (map[int64]string, error) {
	if len(ids) == 0 {
		return map[int64]string{}, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf("SELECT id, path FROM comments WHERE id IN (%s)", strings.Join(placeholders, ","))
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int64]string)
	for rows.Next() {
		var id int64
		var path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		result[id] = path
	}
	return result, nil
}

func (r *Repo) GetByPathPrefix(path string) ([]*models.Comment, error) {
	return r.getList("SELECT id, parent_id, path, body, author, created_at FROM comments WHERE path LIKE ? || '%' ORDER BY path", path)
}

func (r *Repo) getList(query string, args ...interface{}) ([]*models.Comment, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Comment
	for rows.Next() {
		c := &models.Comment{}
		var parentID sql.NullInt64
		var createdAt string
		if err := rows.Scan(&c.ID, &parentID, &c.Path, &c.Body, &c.Author, &createdAt); err != nil {
			return nil, err
		}
		if parentID.Valid {
			c.ParentID = &parentID.Int64
		}
		c.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		list = append(list, c)
	}
	return list, nil
}
