package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/orchestrator/models"
	"github.com/romariotrain/media-platform/internal/orchestrator/service"
)

// Handler обрабатывает HTTP-запросы Orchestrator сервиса
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

// CreateSaga создаёт новую сагу
// POST /sagas
func (h *Handler) CreateSaga(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req CreateSagaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorJSON(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.AssetID == uuid.Nil || req.UserID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "asset_id and user_id are required")
		return
	}

	saga, err := h.svc.CreateSaga(r.Context(), service.CreateSagaRequest{
		AssetID: req.AssetID,
		UserID:  req.UserID,
	})
	if err != nil {
		mapError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toSagaResponse(saga))
}

// GetSaga возвращает сагу по ID
// GET /sagas?id=<uuid>
func (h *Handler) GetSaga(w http.ResponseWriter, r *http.Request) {
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

	saga, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		mapError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toSagaResponse(saga))
}

// ListSagas возвращает список саг пользователя
// GET /sagas/list?user_id=<string>
func (h *Handler) ListSagas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErrorJSON(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeErrorJSON(w, http.StatusBadRequest, "missing user_id query parameter")
		return
	}

	sagas, err := h.svc.ListByUserID(r.Context(), userID)
	if err != nil {
		mapError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toSagaListResponse(sagas))
}

func mapError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrNotFound):
		writeErrorJSON(w, http.StatusNotFound, "not found")
	case errors.Is(err, models.ErrInvalidArgument):
		writeErrorJSON(w, http.StatusBadRequest, "invalid argument")
	case errors.Is(err, models.ErrSagaCompleted):
		writeErrorJSON(w, http.StatusConflict, "saga already completed")
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
