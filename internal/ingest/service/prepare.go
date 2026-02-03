package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/ingest/models"
)

// PrepareUpload создаёт upload session с токеном.
// Вызывается из Kafka consumer при получении commands.ingest.prepare.
// Идемпотентно: если сессия для данного saga_id уже существует, возвращает её.
func (s *Service) PrepareUpload(ctx context.Context, sagaID uuid.UUID, assetID uuid.UUID, userID string) (*models.UploadSession, error) {
	// 1. Валидация
	if sagaID == uuid.Nil || assetID == uuid.Nil || userID == "" {
		return nil, models.ErrInvalidArgument
	}

	// 2. Идемпотентность: проверяем по saga_id
	existing, err := s.repo.FindBySagaID(ctx, sagaID)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		return nil, fmt.Errorf("check existing session: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	// 3. Создаём новую сессию
	now := s.clock()
	session := &models.UploadSession{
		ID:        s.idGen(),
		SagaID:    sagaID,
		AssetID:   assetID,
		UserID:    userID,
		Token:     s.tokenGen(),
		Status:    models.StatusReady,
		ExpiresAt: now.Add(UploadTokenTTL),
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 4. BEGIN TX
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 5. Создаём сессию в БД
	if err := s.repo.CreateTx(ctx, tx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// 6. Добавляем событие в outbox (в той же транзакции)
	event := models.NewIngestReadyEvent(sagaID, session)
	if err := s.outboxRepo.Add(ctx, tx, string(models.EventIngestReady), session.ID.String(), sagaID, event.Payload); err != nil {
		return nil, fmt.Errorf("add event to outbox: %w", err)
	}

	// 7. COMMIT
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return session, nil
}
