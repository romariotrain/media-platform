package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/romariotrain/media-platform/internal/ingest/models"
	"github.com/romariotrain/media-platform/internal/ingest/service"
)

// Handler обрабатывает HTTP-запросы Ingest сервиса
type Handler struct {
	svc *service.Service
}

// New создаёт новый Handler
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

// Health проверяет состояние сервиса
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// GetUploadToken возвращает информацию о токене загрузки
// GET /upload-token?token=abc123
func (h *Handler) GetUploadToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		writeErrorJSON(w, http.StatusBadRequest, "missing token query parameter")
		return
	}

	session, err := h.svc.GetUploadToken(r.Context(), token)
	if err != nil {
		mapError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUploadTokenResponse(session))
}

// Upload принимает файл по токену загрузки
// POST /upload?token=abc123 (multipart/form-data, field "file")
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		writeErrorJSON(w, http.StatusBadRequest, "missing token query parameter")
		return
	}

	// Парсим multipart form (макс 500MB + 10MB overhead)
	if err := r.ParseMultipartForm(service.MaxFileSize + 10<<20); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	session, err := h.svc.UploadFile(
		r.Context(),
		token,
		header.Filename,
		header.Size,
		contentType,
		file,
	)
	if err != nil {
		mapError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toUploadResponse(session))
}

func mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		writeErrorJSON(w, http.StatusNotFound, "not found")
	case errors.Is(err, models.ErrInvalidArgument):
		writeErrorJSON(w, http.StatusBadRequest, "invalid argument")
	case errors.Is(err, models.ErrTokenExpired):
		writeErrorJSON(w, http.StatusGone, "upload token expired")
	case errors.Is(err, models.ErrTokenUsed):
		writeErrorJSON(w, http.StatusConflict, "upload token already used")
	case errors.Is(err, models.ErrFileTooLarge):
		writeErrorJSON(w, http.StatusRequestEntityTooLarge, "file too large")
	case errors.Is(err, models.ErrUnsupportedType):
		writeErrorJSON(w, http.StatusUnsupportedMediaType, "unsupported file type")
	default:
		writeErrorJSON(w, http.StatusInternalServerError, "internal error")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErrorJSON(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
