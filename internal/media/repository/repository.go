package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/media/models"
)

type MediaRepository interface {
	Create(ctx context.Context, m *models.Media) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Media, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.Status) (*models.Media, error)
}
