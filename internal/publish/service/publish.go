package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/publish/models"
)

// PublishMedia публикует обработанные медиафайлы.
// Идемпотентно: если публикация для данного saga_id уже существует и завершена, возвращает её.
func (s *Service) PublishMedia(ctx context.Context, sagaID uuid.UUID, assetID uuid.UUID, sourcePaths *models.SourcePaths) (*models.Publication, error) {
	// 1. Валидация
	if sagaID == uuid.Nil || assetID == uuid.Nil || sourcePaths == nil {
		return nil, models.ErrInvalidArgument
	}

	// 2. Идемпотентность: проверяем по saga_id
	existing, err := s.repo.FindBySagaID(ctx, sagaID)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		return nil, fmt.Errorf("check existing publication: %w", err)
	}
	if existing != nil {
		if existing.Status == models.StatusPublished || existing.Status == models.StatusFailed {
			return existing, nil
		}
		if existing.Status == models.StatusPublishing {
			return existing, nil
		}
	}

	// 3. Создаём публикацию со статусом publishing
	now := s.clock()
	pub := &models.Publication{
		ID:          s.idGen(),
		SagaID:      sagaID,
		AssetID:     assetID,
		Status:      models.StatusPublishing,
		SourcePaths: sourcePaths,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// 4. Сохраняем в БД (отдельная транзакция)
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	if err := s.repo.CreateTx(ctx, tx, pub); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("create publication: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create tx: %w", err)
	}

	// 5. Выполняем публикацию (копирование файлов, вне транзакции)
	result, publishErr := s.publisher.Publish(assetID.String(), sourcePaths)

	// 6. Обновляем публикацию с результатом
	publishedAt := s.clock()
	pub.UpdatedAt = publishedAt

	var event models.Event

	if publishErr != nil {
		pub.Status = models.StatusFailed
		errMsg := publishErr.Error()
		pub.ErrorMessage = &errMsg
		event = models.NewPublishFailedEvent(sagaID, pub, errMsg)
	} else {
		pub.Status = models.StatusPublished
		pub.PublishedPaths = result.PublishedPaths
		pub.PublicURLs = result.PublicURLs
		pub.PublishedAt = &publishedAt
		event = models.NewPublishSucceededEvent(sagaID, pub)
	}

	// 7. Сохраняем результат и событие в одной транзакции
	tx2, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx2: %w", err)
	}
	defer tx2.Rollback()

	if err := s.repo.UpdateTx(ctx, tx2, pub); err != nil {
		return nil, fmt.Errorf("update publication: %w", err)
	}

	if err := s.outboxRepo.Add(ctx, tx2, string(event.Type), pub.ID.String(), sagaID, event.Payload); err != nil {
		return nil, fmt.Errorf("add event to outbox: %w", err)
	}

	if err := tx2.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx2: %w", err)
	}

	return pub, nil
}

// GetByID возвращает публикацию по ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.Publication, error) {
	if id == uuid.Nil {
		return nil, models.ErrInvalidArgument
	}
	return s.repo.GetByID(ctx, id)
}

// ListByAssetID возвращает все публикации для указанного asset
func (s *Service) ListByAssetID(ctx context.Context, assetID uuid.UUID) ([]*models.Publication, error) {
	if assetID == uuid.Nil {
		return nil, models.ErrInvalidArgument
	}
	return s.repo.ListByAssetID(ctx, assetID)
}
