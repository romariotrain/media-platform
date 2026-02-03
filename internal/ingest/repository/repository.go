package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/romariotrain/media-platform/internal/ingest/models"
)

// UploadSessionRepository определяет интерфейс для работы с сессиями загрузки
type UploadSessionRepository interface {
	// Чтение
	GetByID(ctx context.Context, id uuid.UUID) (*models.UploadSession, error)
	GetByToken(ctx context.Context, token string) (*models.UploadSession, error)
	FindBySagaID(ctx context.Context, sagaID uuid.UUID) (*models.UploadSession, error)

	// Транзакции
	BeginTx(ctx context.Context) (*sqlx.Tx, error)

	// Запись (в транзакции)
	CreateTx(ctx context.Context, tx *sqlx.Tx, session *models.UploadSession) error
	UpdateStatusTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, status models.UploadStatus) error
	UpdateUploadInfoTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, fileName string, fileSize int64, contentType string, storagePath string) error
}
