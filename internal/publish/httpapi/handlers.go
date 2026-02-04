package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/publish/models"
	"github.com/romariotrain/media-platform/internal/publish/service"
)

// Handler обрабатывает HTTP-запросы Publish сервиса
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

// GetPublication возвращает публикацию по ID
// GET /publications?id=<uuid>
func (h *Handler) GetPublication(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		writeErrorJSON(w, http.StatusBadRequest, "missing id query parameter")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid id format")
		return
	}

	pub, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toPublicationResponse(pub))
}

// ListPublications возвращает список публикаций по asset_id
// GET /publications/list?asset_id=<uuid>
func (h *Handler) ListPublications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	assetIDStr := r.URL.Query().Get("asset_id")
	if assetIDStr == "" {
		writeErrorJSON(w, http.StatusBadRequest, "missing asset_id query parameter")
		return
	}

	assetID, err := uuid.Parse(assetIDStr)
	if err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid asset_id format")
		return
	}

	pubs, err := h.svc.ListByAssetID(r.Context(), assetID)
	if err != nil {
		mapError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toPublicationListResponse(pubs))
}

func mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		writeErrorJSON(w, http.StatusNotFound, "not found")
	case errors.Is(err, models.ErrInvalidArgument):
		writeErrorJSON(w, http.StatusBadRequest, "invalid argument")
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
