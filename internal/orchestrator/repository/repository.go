package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/romariotrain/media-platform/internal/orchestrator/models"
)

// SagaRepository определяет интерфейс для работы с сагами
type SagaRepository interface {
	// Чтение
	GetByID(ctx context.Context, id uuid.UUID) (*models.Saga, error)
	ListByUserID(ctx context.Context, userID string) ([]*models.Saga, error)
	ListByAssetID(ctx context.Context, assetID uuid.UUID) ([]*models.Saga, error)

	// Транзакции
	BeginTx(ctx context.Context) (*sqlx.Tx, error)

	// Запись (в транзакции)
	CreateTx(ctx context.Context, tx *sqlx.Tx, saga *models.Saga) error
	UpdateTx(ctx context.Context, tx *sqlx.Tx, saga *models.Saga) error
}
