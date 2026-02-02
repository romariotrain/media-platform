package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/quota/models"
)

// Tx представляет транзакцию БД
type Tx interface {
	Commit() error
	Rollback() error
}

// QuotaRepository интерфейс для работы с квотами
type QuotaRepository interface {
	// Транзакции
	BeginTx(ctx context.Context) (Tx, error)

	// Работа с резервациями квот (quotas table)

	// FindByAssetID возвращает квоту по ID ассета (для проверки идемпотентности)
	// Возвращает ErrNotFound если квота не найдена
	FindByAssetID(ctx context.Context, assetID uuid.UUID) (*models.Quota, error)

	// Create создаёт новую резервацию квоты в транзакции
	CreateInTx(ctx context.Context, tx Tx, quota *models.Quota) error

	// UpdateStatus обновляет статус квоты (reserved -> released)
	UpdateStatusInTx(ctx context.Context, tx Tx, quotaID uuid.UUID, status models.QuotaStatus, releasedAt *sql.NullTime) error

	// Работа с лимитами пользователей (user_quotas table)

	// GetUserQuotaForUpdate получает квоту пользователя с блокировкой строки (SELECT FOR UPDATE)
	// Должен вызываться внутри транзакции для защиты от race conditions
	GetUserQuotaForUpdateInTx(ctx context.Context, tx Tx, userID string, quotaType models.QuotaType) (*models.UserQuota, error)

	// IncrementUsedAmount увеличивает used_amount в транзакции
	// Предполагается что проверка лимита уже сделана через GetUserQuotaForUpdate
	IncrementUsedAmountInTx(ctx context.Context, tx Tx, userID string, quotaType models.QuotaType, amount int64) error

	// DecrementUsedAmount уменьшает used_amount в транзакции
	DecrementUsedAmountInTx(ctx context.Context, tx Tx, userID string, quotaType models.QuotaType, amount int64) error

	// GetUserQuota получает текущее состояние квоты пользователя (без блокировки)
	// Для read-only операций
	GetUserQuota(ctx context.Context, userID string, quotaType models.QuotaType) (*models.UserQuota, error)
}
