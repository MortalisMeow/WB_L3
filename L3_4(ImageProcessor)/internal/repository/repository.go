package repository

import (
	"database/sql"
	"time"

	"imageprocessor/internal/models"
)

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) Create(filename string) (*models.Image, error) {
	res, err := r.db.Exec("INSERT INTO images (filename) VALUES (?)", filename)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetByID(id)
}

func (r *Repo) GetByID(id int64) (*models.Image, error) {
	img := &models.Image{}
	var createdAt string
	err := r.db.QueryRow("SELECT id, filename, status, created_at FROM images WHERE id = ?", id).
		Scan(&img.ID, &img.Filename, &img.Status, &createdAt)
	if err != nil {
		return nil, err
	}
	img.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
	return img, nil
}

func (r *Repo) UpdateStatus(id int64, status string) error {
	_, err := r.db.Exec("UPDATE images SET status = ? WHERE id = ?", status, id)
	return err
}

func (r *Repo) Delete(id int64) error {
	_, err := r.db.Exec("DELETE FROM images WHERE id = ?", id)
	return err
}

func (r *Repo) List() ([]*models.Image, error) {
	rows, err := r.db.Query("SELECT id, filename, status, created_at FROM images ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*models.Image
	for rows.Next() {
		img := &models.Image{}
		var createdAt string
		if err := rows.Scan(&img.ID, &img.Filename, &img.Status, &createdAt); err != nil {
			return nil, err
		}
		img.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdAt)
		list = append(list, img)
	}
	return list, nil
}
