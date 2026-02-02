package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/quota/models"
)

// ReleaseQuota освобождает зарезервированную квоту
// Идемпотентна: повторный вызов для уже освобождённой квоты вернёт success
func (s *Service) ReleaseQuota(ctx context.Context, assetID uuid.UUID) (*models.Quota, error) {
	// Валидация
	if assetID == uuid.Nil {
		return nil, models.ErrInvalidArgument
	}

	// 1. Находим квоту по assetID
	quota, err := s.repo.FindByAssetID(ctx, assetID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			// Квоты нет - возможно она никогда не резервировалась
			return nil, fmt.Errorf("quota for asset %s not found: %w", assetID, err)
		}
		return nil, fmt.Errorf("find quota: %w", err)
	}

	// 2. Проверка идемпотентности: если квота уже освобождена - возвращаем success
	if quota.Status == models.QuotaReleased {
		return quota, nil
	}

	// 3. Проверяем что квота в правильном статусе для освобождения
	if quota.Status != models.QuotaReserved {
		return nil, fmt.Errorf("cannot release quota with status %s", quota.Status)
	}

	// 4. Начинаем транзакцию
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 5. Обновляем статус квоты на released
	now := s.clock()
	releasedAt := sql.NullTime{Time: now, Valid: true}

	if err := s.repo.UpdateStatusInTx(ctx, tx, quota.ID, models.QuotaReleased, &releasedAt); err != nil {
		return nil, fmt.Errorf("update quota status: %w", err)
	}

	// 6. Уменьшаем used_amount (возвращаем квоту пользователю)
	if err := s.repo.DecrementUsedAmountInTx(ctx, tx, quota.UserID, quota.Type, quota.Amount); err != nil {
		return nil, fmt.Errorf("decrement used amount: %w", err)
	}

	// 7. Добавляем событие в outbox
	eventPayload := map[string]interface{}{
		"quota_id":    quota.ID,
		"asset_id":    quota.AssetID,
		"user_id":     quota.UserID,
		"amount":      quota.Amount,
		"released_at": now,
	}

	if err := s.outboxRepo.Add(ctx, tx, string(models.EventQuotaReleased), quota.SagaID, eventPayload); err != nil {
		return nil, fmt.Errorf("add event to outbox: %w", err)
	}

	// 8. Коммитим транзакцию
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	// Обновляем локальную копию
	quota.Status = models.QuotaReleased
	quota.ReleasedAt = &now
	quota.UpdatedAt = now

	return quota, nil
}
