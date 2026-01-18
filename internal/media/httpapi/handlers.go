package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/romariotrain/media-platform/internal/media/models"
	"github.com/romariotrain/media-platform/internal/media/service"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) CreateMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	defer r.Body.Close()

	var req CreateMediaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid json body")
		return
	}

	m, err := h.svc.CreateMedia(r.Context(), req.Type, req.Source)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrInvalidArgument):
			writeErrorJSON(w, http.StatusBadRequest, "invalid argument")
		case errors.Is(err, models.ErrConflict):
			writeErrorJSON(w, http.StatusConflict, "conflict")
		default:
			writeErrorJSON(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, toMediaResponse(m))
}

func (h *Handler) GetMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// ожидаем path вида /media/{id}
	idStr := strings.TrimPrefix(r.URL.Path, "/media/")
	if idStr == "" || idStr == r.URL.Path {
		writeErrorJSON(w, http.StatusBadRequest, "missing id")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		panic(err)
	}

	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid id")
		return
	}

	m, err := h.svc.GetMedia(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, models.ErrNotFound):
			writeErrorJSON(w, http.StatusNotFound, "not found")
		case errors.Is(err, models.ErrInvalidArgument):
			writeErrorJSON(w, http.StatusBadRequest, "invalid argument")
		default:
			writeErrorJSON(w, http.StatusInternalServerError, "internal error: "+err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, toMediaResponse(m))
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErrorJSON(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func toMediaResponse(m *models.Media) MediaResponse {
	return MediaResponse{
		ID:        m.ID,
		Status:    string(m.Status),
		Type:      m.Type,
		Source:    m.Source,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}

func (h *Handler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим ID из URL: /media/{id}/status
	path := strings.TrimPrefix(r.URL.Path, "/media/")
	idStr := strings.TrimSuffix(path, "/status")

	mediaID, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Парсим body
	var req struct {
		Status models.Status `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	// Вызываем сервис
	media, err := h.svc.ChangeStatus(r.Context(), mediaID, req.Status)
	if err != nil {
		// TODO: обработка разных ошибок (404, validation)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Возвращаем результат
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(media)
}
