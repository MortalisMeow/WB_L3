package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"shortener/internal/service"
	"strings"
)

type Handler struct {
	service service.URLService
}

func NewHandler(srv service.URLService) *Handler {
	return &Handler{
		service: srv,
	}
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	ShortURL string `json:"short_url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	//читаем тело запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Can't read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close() //закрываем поток, освобождаем ресурс

	//парсинг JSON
	var req ShortenRequest

	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	//валидация JSON
	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	//вызов сервиса
	shortURL, err := h.service.ShortenURL(r.Context(), req.URL)
	if err != nil {
		http.Error(w, "Error while shorten url", http.StatusBadRequest)
		return
	}

	resp := ShortenResponse{
		ShortURL: shortURL,
	}

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	shortCode := strings.TrimPrefix(r.URL.Path, "/s/")

	if shortCode == "" {
		h.writeError(w, http.StatusBadRequest, "Short code is required")
		return
	}

	originalURL, err := h.service.GetOriginalURL(r.Context(), shortCode)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "Short URL not found")
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
	shortCode := strings.TrimPrefix(r.URL.Path, "/analytics/")

	if shortCode == "" {
		h.writeError(w, http.StatusBadRequest, "Short code is required")
		return
	}

	analytics, err := h.service.GetAnalytics(r.Context(), shortCode)
	if err != nil {
		h.writeError(w, http.StatusNotFound, "Short URL not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(analytics)
}

func (h *Handler) writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
