package delivery

import (
	"delayed-notifier/internal/usecase"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type Handler struct {
	uc *usecase.NotificationUseCase
}

func NewHandler(uc *usecase.NotificationUseCase) *Handler {
	return &Handler{uc: uc}
}

func (h *Handler) Routes(r *mux.Router) {
	r.HandleFunc("/notify", h.CreateNotification).Methods("POST")
	r.HandleFunc("/notify/{id}", h.GetNotification).Methods("GET")
	r.HandleFunc("/notify/{id}", h.CancelNotification).Methods("DELETE")
}

type createRequest struct {
	Receiver    string `json:"receiver"`
	Topic       string `json:"topic"`
	ScheduledAt string `json:"scheduled_at"`
}

// POST
func (h *Handler) CreateNotification(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		http.Error(w, "invalid time format", http.StatusBadRequest)
		return
	}

	notification, err := h.uc.CreateNotification(usecase.CreateNotificationRequest{
		Receiver:    req.Receiver,
		Topic:       req.Topic,
		ScheduledAt: scheduledAt,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(notification)
}

// GET
func (h *Handler) GetNotification(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	notification, err := h.uc.GetNotification(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}

// DELETE
func (h *Handler) CancelNotification(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	if err := h.uc.CancelNotification(id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
