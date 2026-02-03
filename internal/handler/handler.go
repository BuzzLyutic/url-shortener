package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/BuzzLyutic/url-shortener/internal/service"
	"github.com/BuzzLyutic/url-shortener/internal/shortcode"
)

type Handler struct {
	service *service.Shortener
	logger  *slog.Logger
}

func New(svc *service.Shortener, logger *slog.Logger) *Handler {
	return &Handler{
		service: svc,
		logger:  logger,
	}
}

// Метод регистрирует все пути к данному мультиплексеру
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// API эндпоинты
	mux.HandleFunc("POST /api/shorten", h.Shorten)
	mux.HandleFunc("GET /{code}", h.Redirect)
	mux.HandleFunc("GET /health", h.Health)
}

// Обрабатывает запросы POST /api/shorten
func (h *Handler) Shorten(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid json", "Invalid JSON body")
		return
	}

	result, err := h.service.Shorten(r.Context(), req.URL)
	if err != nil {
		h.handleServiceError(w, err)
		return
	}

	status := http.StatusCreated
	if !result.IsNew {
		status = http.StatusOK
	}

	h.writeJSON(w, status, ShortenResponse{
		ShortURL:    result.ShortURL,
		OriginalURL: result.OriginalURL,
		ExpiresAt:   result.ExpiresAt,
	})
}

// Обрабатывает запросы GET /{code}
func (h *Handler) Redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")

	// Валидация формата кода
	if !shortcode.IsValid(code) {
		h.writeError(w, http.StatusNotFound, "not_found", "Short URL not found")
		return
	}
	originalURL, err := h.service.Resolve(r.Context(), code)
	if err != nil {
		if errors.Is(err, service.ErrCodeNotFound) {
			h.writeError(w, http.StatusNotFound, "not_found", "Short URL not found")
			return
		}
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
		return
	}
	http.Redirect(w, r, originalURL, http.StatusMovedPermanently)
}

// Обрабатывает GET /health
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Данный метод отображает ошибки сервиса на HTTP ответы
func (h *Handler) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrEmptyURL):
		h.writeError(w, http.StatusBadRequest, "empty_url", "URL cannot be empty")
	case errors.Is(err, service.ErrInvalidURL):
		h.writeError(w, http.StatusBadRequest, "invalid_url", "Invalid URL format.")
	case errors.Is(err, service.ErrTooManyCollisions):
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Failed to generate short URL")
	default:
		h.logger.Error("unexpected error", slog.Any("error", err))
		h.writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

// Метод записывает JSON ответ
func (h *Handler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("failed to encode response", slog.Any("error", err))
	}
}

// Метод, записывающий сообщение об ошибке
func (h *Handler) writeError(w http.ResponseWriter, status int, errCode, message string) {
	h.writeJSON(w, status, ErrorResponse{
		Error:   errCode,
		Message: message,
	})
}
