package httpapi

import (
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/media/models"
)

type CreateMediaRequest struct {
	Type   models.MediaType `json:"type"`
	Source string           `json:"source"`
}

type MediaResponse struct {
	ID        uuid.UUID        `json:"id"`
	Status    string           `json:"status"`
	Type      models.MediaType `json:"type"`
	Source    string           `json:"source"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}
