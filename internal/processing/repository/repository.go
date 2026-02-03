package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/romariotrain/media-platform/internal/processing/models"
)

// ProcessingTaskRepository определяет интерфейс для работы с задачами обработки
type ProcessingTaskRepository interface {
	// Чтение
	GetByID(ctx context.Context, id uuid.UUID) (*models.ProcessingTask, error)
	FindBySagaID(ctx context.Context, sagaID uuid.UUID) (*models.ProcessingTask, error)

	// Транзакции
	BeginTx(ctx context.Context) (*sqlx.Tx, error)

	// Запись (в транзакции)
	CreateTx(ctx context.Context, tx *sqlx.Tx, task *models.ProcessingTask) error
	UpdateTx(ctx context.Context, tx *sqlx.Tx, task *models.ProcessingTask) error
}
