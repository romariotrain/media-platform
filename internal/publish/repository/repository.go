package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/romariotrain/media-platform/internal/publish/models"
)

// PublicationRepository определяет интерфейс для работы с публикациями
type PublicationRepository interface {
	// Чтение
	GetByID(ctx context.Context, id uuid.UUID) (*models.Publication, error)
	FindBySagaID(ctx context.Context, sagaID uuid.UUID) (*models.Publication, error)
	ListByAssetID(ctx context.Context, assetID uuid.UUID) ([]*models.Publication, error)

	// Транзакции
	BeginTx(ctx context.Context) (*sqlx.Tx, error)

	// Запись (в транзакции)
	CreateTx(ctx context.Context, tx *sqlx.Tx, pub *models.Publication) error
	UpdateTx(ctx context.Context, tx *sqlx.Tx, pub *models.Publication) error
}
