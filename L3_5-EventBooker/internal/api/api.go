package api

import (
	"strconv"

	"eventbooker/internal/svc"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *svc.Service
}

func New(svc *svc.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) CreateEvent(c *gin.Context) {
	var req struct {
		Name  string `json:"name"`
		Date  string `json:"date"`
		Seats int    `json:"seats"`
	}
	c.ShouldBindJSON(&req)
	e, err := h.svc.CreateEvent(req.Name, req.Date, req.Seats)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(201, e)
}

func (h *Handler) ListEvents(c *gin.Context) {
	events, _ := h.svc.ListEvents()
	c.JSON(200, events)
}

func (h *Handler) GetEvent(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	event, err := h.svc.GetEvent(id)
	if err != nil {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}
	bookings, _ := h.svc.GetBookings(id)
	c.JSON(200, gin.H{"event": event, "bookings": bookings})
}

func (h *Handler) Book(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		UserName string `json:"user_name"`
	}
	c.ShouldBindJSON(&req)
	b, err := h.svc.Book(id, req.UserName)
	if err != nil {
		c.JSON(409, gin.H{"error": "no seats"})
		return
	}
	c.JSON(201, b)
}

func (h *Handler) Confirm(c *gin.Context) {
	var req struct {
		BookingID int64 `json:"booking_id"`
	}
	c.ShouldBindJSON(&req)
	h.svc.Confirm(req.BookingID)
	c.JSON(200, gin.H{"status": "ok"})
}

func (h *Handler) DeleteEvent(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.svc.DeleteEvent(id)
	c.Status(204)
}
