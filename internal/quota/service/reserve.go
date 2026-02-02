package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/quota/models"
)

// ReserveQuota резервирует квоту для ассета
// Идемпотентна: повторный вызов для того же assetID вернёт существующую квоту
func (s *Service) ReserveQuota(ctx context.Context, sagaID uuid.UUID, userID string, assetID uuid.UUID, quotaType models.QuotaType, amount int64) (*models.Quota, error) {
	// Валидация входных параметров
	if sagaID == uuid.Nil || userID == "" || assetID == uuid.Nil || amount <= 0 {
		return nil, models.ErrInvalidArgument
	}

	// 1. Проверка идемпотентности: есть ли уже квота для этого assetID?
	existingQuota, err := s.repo.FindByAssetID(ctx, assetID)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		return nil, fmt.Errorf("check existing quota: %w", err)
	}

	// Если квота уже существует и она зарезервирована - возвращаем её (идемпотентность)
	if existingQuota != nil && existingQuota.Status == models.QuotaReserved {
		return existingQuota, nil
	}

	// Если квота существует но освобождена - это странная ситуация
	// (значит сага раньше прошла и откатилась, теперь запущена заново)
	// Для простоты вернём ошибку, но можно обсудить другую логику
	if existingQuota != nil && existingQuota.Status == models.QuotaReleased {
		return nil, fmt.Errorf("quota for asset %s was already released", assetID)
	}

	// 2. Начинаем транзакцию
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() // откат если не сделаем Commit

	// 3. Получаем текущее состояние квоты пользователя с блокировкой (SELECT FOR UPDATE)
	userQuota, err := s.repo.GetUserQuotaForUpdateInTx(ctx, tx, userID, quotaType)
	if err != nil {
		return nil, fmt.Errorf("get user quota: %w", err)
	}

	// 4. Проверяем достаточно ли свободной квоты
	if !userQuota.CanReserve(amount) {
		return nil, fmt.Errorf("%w: requested %d, available %d",
			models.ErrInsufficientQuota, amount, userQuota.AvailableAmount())
	}

	// 5. Увеличиваем used_amount
	if err := s.repo.IncrementUsedAmountInTx(ctx, tx, userID, quotaType, amount); err != nil {
		return nil, fmt.Errorf("increment used amount: %w", err)
	}

	// 6. Создаём запись резервации
	now := s.clock()
	quota := &models.Quota{
		ID:         s.idGen(),
		UserID:     userID,
		AssetID:    assetID,
		SagaID:     sagaID,
		Type:       quotaType,
		Amount:     amount,
		Status:     models.QuotaReserved,
		ReservedAt: now,
		ExpiresAt:  now.Add(QuotaReservationTTL),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.CreateInTx(ctx, tx, quota); err != nil {
		return nil, fmt.Errorf("create quota: %w", err)
	}

	// 7. Добавляем событие в outbox (в той же транзакции!)
	eventPayload := map[string]interface{}{
		"quota_id":   quota.ID,
		"asset_id":   quota.AssetID,
		"user_id":    quota.UserID,
		"type":       quota.Type,
		"amount":     quota.Amount,
		"expires_at": quota.ExpiresAt,
	}

	if err := s.outboxRepo.Add(ctx, tx, string(models.EventQuotaReserved), quota.SagaID, eventPayload); err != nil {
		return nil, fmt.Errorf("add event to outbox: %w", err)
	}

	// 8. Коммитим транзакцию (атомарно!)
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return quota, nil
}
