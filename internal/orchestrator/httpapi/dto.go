package httpapi

import (
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/orchestrator/models"
)

// CreateSagaRequest — тело запроса для создания саги
type CreateSagaRequest struct {
	AssetID uuid.UUID `json:"asset_id"`
	UserID  string    `json:"user_id"`
}

// SagaResponse — ответ с информацией о саге
type SagaResponse struct {
	ID           uuid.UUID          `json:"id"`
	AssetID      uuid.UUID          `json:"asset_id"`
	UserID       string             `json:"user_id"`
	Status       string             `json:"status"`
	CurrentStep  string             `json:"current_step"`
	Payload      *models.SagaPayload `json:"payload,omitempty"`
	ErrorMessage *string            `json:"error_message,omitempty"`
	StartedAt    *time.Time         `json:"started_at,omitempty"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
	CreatedAt    time.Time          `json:"created_at"`
}

func toSagaResponse(saga *models.Saga) SagaResponse {
	return SagaResponse{
		ID:           saga.ID,
		AssetID:      saga.AssetID,
		UserID:       saga.UserID,
		Status:       string(saga.Status),
		CurrentStep:  string(saga.CurrentStep),
		Payload:      saga.Payload,
		ErrorMessage: saga.ErrorMessage,
		StartedAt:    saga.StartedAt,
		CompletedAt:  saga.CompletedAt,
		CreatedAt:    saga.CreatedAt,
	}
}

func toSagaListResponse(sagas []*models.Saga) []SagaResponse {
	result := make([]SagaResponse, 0, len(sagas))
	for _, saga := range sagas {
		result = append(result, toSagaResponse(saga))
	}
	return result
}
