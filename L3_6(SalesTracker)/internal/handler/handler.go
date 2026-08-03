package handler

import (
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"

	"salestracker/internal/models"
	"salestracker/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Create(c *gin.Context) {
	var req models.CreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	item, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context(), h.filter(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req models.UpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	item, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) Analytics(c *gin.Context) {
	f := h.filter(c)
	stats, err := h.svc.Analytics(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	daily, err := h.svc.DailyTotals(c.Request.Context(), f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"stats":  stats,
		"daily":  daily,
	})
}

func (h *Handler) ExportCSV(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context(), h.filter(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=items.csv")

	w := csv.NewWriter(c.Writer)
	w.Write([]string{"id", "type", "amount", "category", "occurred_at", "created_at"})
	for _, item := range items {
		w.Write([]string{
			strconv.FormatInt(item.ID, 10),
			item.Type,
			strconv.FormatFloat(item.Amount, 'f', 2, 64),
			item.Category,
			item.OccurredAt.Format("2006-01-02"),
			item.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	w.Flush()
}

func (h *Handler) filter(c *gin.Context) models.ItemFilter {
	return models.ItemFilter{
		From:     c.Query("from"),
		To:       c.Query("to"),
		Type:     c.Query("type"),
		Category: c.Query("category"),
	}
}

func (h *Handler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, models.ErrInvalidType),
		errors.Is(err, models.ErrInvalidAmount),
		errors.Is(err, models.ErrInvalidDate),
		errors.Is(err, models.ErrEmptyCategory):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
