package svc

import (
	"time"

	"eventbooker/internal/repo"
)

type Service struct {
	repo *repo.Repo
}

func New(repo *repo.Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateEvent(name, date string, seats int) (*repo.Event, error) {
	return s.repo.CreateEvent(name, date, seats)
}

func (s *Service) GetEvent(id int64) (*repo.Event, error) {
	return s.repo.GetEvent(id)
}

func (s *Service) ListEvents() ([]repo.Event, error) {
	return s.repo.ListEvents()
}

func (s *Service) DeleteEvent(id int64) error {
	return s.repo.DeleteEvent(id)
}

func (s *Service) Book(eventID int64, userName string) (*repo.Booking, error) {
	if userName == "" {
		userName = "Anonymous"
	}
	return s.repo.Book(eventID, userName)
}

func (s *Service) Confirm(bookingID int64) error {
	return s.repo.ConfirmBooking(bookingID)
}

func (s *Service) GetBookings(eventID int64) ([]repo.Booking, error) {
	return s.repo.GetBookings(eventID)
}

func (s *Service) CancelWorker() {
	for {
		time.Sleep(30 * time.Second)
		s.repo.CancelExpired(5)
	}
}
