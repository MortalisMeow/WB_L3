package repo

import (
	"database/sql"
)

type Event struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Date   string `json:"date"`
	Seats  int    `json:"seats"`
	Booked int    `json:"booked"`
}

type Booking struct {
	ID       int64  `json:"id"`
	EventID  int64  `json:"event_id"`
	UserName string `json:"user_name"`
	Status   string `json:"status"`
}

type Repo struct {
	db *sql.DB
}

func New(db *sql.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) CreateEvent(name, date string, seats int) (*Event, error) {
	res, err := r.db.Exec("INSERT INTO events (name, date, seats) VALUES (?, ?, ?)", name, date, seats)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return r.GetEvent(id)
}

func (r *Repo) GetEvent(id int64) (*Event, error) {
	e := &Event{}
	err := r.db.QueryRow("SELECT id, name, date, seats, booked FROM events WHERE id = ?", id).
		Scan(&e.ID, &e.Name, &e.Date, &e.Seats, &e.Booked)
	return e, err
}

func (r *Repo) ListEvents() ([]Event, error) {
	rows, err := r.db.Query("SELECT id, name, date, seats, booked FROM events ORDER BY id DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Event
	for rows.Next() {
		var e Event
		rows.Scan(&e.ID, &e.Name, &e.Date, &e.Seats, &e.Booked)
		list = append(list, e)
	}
	return list, nil
}

func (r *Repo) DeleteEvent(id int64) error {
	r.db.Exec("DELETE FROM bookings WHERE event_id = ?", id)
	_, err := r.db.Exec("DELETE FROM events WHERE id = ?", id)
	return err
}

func (r *Repo) Book(eventID int64, userName string) (*Booking, error) {
	tx, _ := r.db.Begin()

	var booked, seats int
	tx.QueryRow("SELECT booked, seats FROM events WHERE id = ?", eventID).Scan(&booked, &seats)
	if booked >= seats {
		tx.Rollback()
		return nil, sql.ErrNoRows
	}

	tx.Exec("UPDATE events SET booked = booked + 1 WHERE id = ?", eventID)
	res, _ := tx.Exec("INSERT INTO bookings (event_id, user_name) VALUES (?, ?)", eventID, userName)
	tx.Commit()

	id, _ := res.LastInsertId()
	return r.GetBooking(id)
}

func (r *Repo) GetBooking(id int64) (*Booking, error) {
	b := &Booking{}
	err := r.db.QueryRow("SELECT id, event_id, user_name, status FROM bookings WHERE id = ?", id).
		Scan(&b.ID, &b.EventID, &b.UserName, &b.Status)
	return b, err
}

func (r *Repo) ConfirmBooking(id int64) error {
	_, err := r.db.Exec("UPDATE bookings SET status = 'confirmed' WHERE id = ? AND status = 'pending'", id)
	return err
}

func (r *Repo) GetBookings(eventID int64) ([]Booking, error) {
	rows, err := r.db.Query("SELECT id, event_id, user_name, status FROM bookings WHERE event_id = ? ORDER BY id DESC", eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Booking
	for rows.Next() {
		var b Booking
		rows.Scan(&b.ID, &b.EventID, &b.UserName, &b.Status)
		list = append(list, b)
	}
	return list, nil
}

func (r *Repo) CancelExpired(minutes int) error {
	rows, _ := r.db.Query(`
		SELECT id, event_id FROM bookings 
		WHERE status = 'pending' 
		AND datetime(created_at, '+' || ? || ' minutes') <= datetime('now')
	`, minutes)
	defer rows.Close()

	for rows.Next() {
		var id, eventID int64
		rows.Scan(&id, &eventID)
		r.db.Exec("UPDATE bookings SET status = 'cancelled' WHERE id = ?", id)
		r.db.Exec("UPDATE events SET booked = booked - 1 WHERE id = ? AND booked > 0", eventID)
	}
	return nil
}
