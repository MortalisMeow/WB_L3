package handler

import (
	"encoding/csv"
	"errors"
	"net/http"
	"strconv"

	"warehousecontrol/internal/auth"
	"warehousecontrol/internal/middleware"
	"warehousecontrol/internal/models"
	"warehousecontrol/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.Service
	jwt *auth.JWT
}

func New(svc *service.Service, jwt *auth.JWT) *Handler {
	return &Handler{svc: svc, jwt: jwt}
}

func (h *Handler) Login(c *gin.Context) {
	var req models.LoginInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	username, role, _, err := h.svc.Login(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, err := h.jwt.Issue(username, role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.LoginResponse{
		Token:    token,
		Username: username,
		Role:     role,
	})
}

func (h *Handler) Create(c *gin.Context) {
	var req models.CreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	claims := middleware.GetClaims(c)
	item, err := h.svc.Create(c.Request.Context(), claims.Username, req)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

func (h *Handler) List(c *gin.Context) {
	items, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (h *Handler) Get(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	item, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	var req models.UpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	claims := middleware.GetClaims(c)
	item, err := h.svc.Update(c.Request.Context(), claims.Username, id, req)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	claims := middleware.GetClaims(c)
	if err := h.svc.Delete(c.Request.Context(), claims.Username, id); err != nil {
		h.writeErr(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ItemHistory(c *gin.Context) {
	itemID, err := parseID(c)
	if err != nil {
		return
	}
	entries, err := h.svc.ItemHistory(c.Request.Context(), itemID, h.historyFilter(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *Handler) History(c *gin.Context) {
	entries, err := h.svc.History(c.Request.Context(), h.historyFilter(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entries)
}

func (h *Handler) HistoryDiff(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	diff, err := h.svc.HistoryDiff(c.Request.Context(), id)
	if err != nil {
		h.writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, diff)
}

func (h *Handler) ExportHistoryCSV(c *gin.Context) {
	entries, err := h.svc.History(c.Request.Context(), h.historyFilter(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=history.csv")

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"id", "item_id", "action", "changed_by", "changed_at", "old_data", "new_data"})
	for _, e := range entries {
		_ = w.Write([]string{
			strconv.FormatInt(e.ID, 10),
			strconv.FormatInt(e.ItemID, 10),
			e.Action,
			e.ChangedBy,
			e.ChangedAt.Format("2006-01-02 15:04:05"),
			string(e.OldData),
			string(e.NewData),
		})
	}
	w.Flush()
}

func (h *Handler) historyFilter(c *gin.Context) models.HistoryFilter {
	itemID, _ := strconv.ParseInt(c.Query("item_id"), 10, 64)
	return models.HistoryFilter{
		ItemID: itemID,
		From:   c.Query("from"),
		To:     c.Query("to"),
		User:   c.Query("user"),
		Action: c.Query("action"),
	}
}

func parseID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return 0, err
	}
	return id, nil
}

func (h *Handler) writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, models.ErrForbidden):
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	case errors.Is(err, models.ErrInvalidName),
		errors.Is(err, models.ErrInvalidSKU),
		errors.Is(err, models.ErrInvalidQty),
		errors.Is(err, models.ErrInvalidPrice),
		errors.Is(err, models.ErrDuplicateSKU):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
