package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/processing/models"
)

// ProcessMedia обрабатывает медиафайл.
// Идемпотентно: если задача для данного saga_id уже существует и завершена, возвращает её.
func (s *Service) ProcessMedia(ctx context.Context, sagaID uuid.UUID, assetID uuid.UUID, inputPath string) (*models.ProcessingTask, error) {
	// 1. Валидация
	if sagaID == uuid.Nil || assetID == uuid.Nil || inputPath == "" {
		return nil, models.ErrInvalidArgument
	}

	// 2. Идемпотентность: проверяем по saga_id
	existing, err := s.repo.FindBySagaID(ctx, sagaID)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		return nil, fmt.Errorf("check existing task: %w", err)
	}
	if existing != nil {
		// Если уже завершена (успешно или с ошибкой) — возвращаем
		if existing.Status == models.StatusCompleted || existing.Status == models.StatusFailed {
			return existing, nil
		}
		// Если ещё в процессе — тоже возвращаем (не запускаем повторно)
		if existing.Status == models.StatusProcessing {
			return existing, nil
		}
	}

	// 3. Создаём задачу со статусом processing
	now := s.clock()
	task := &models.ProcessingTask{
		ID:        s.idGen(),
		SagaID:    sagaID,
		AssetID:   assetID,
		Status:    models.StatusProcessing,
		InputPath: inputPath,
		StartedAt: &now,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// 4. Сохраняем задачу в БД (отдельная транзакция)
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}

	if err := s.repo.CreateTx(ctx, tx, task); err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("create task: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create tx: %w", err)
	}

	// 5. Выполняем обработку (вне транзакции — долгая операция)
	result, processErr := s.processor.Process(ctx, inputPath, task.ID.String())

	// 6. Обновляем задачу с результатом
	completedAt := s.clock()
	task.CompletedAt = &completedAt
	task.UpdatedAt = completedAt

	var event models.Event

	if processErr != nil {
		// Обработка не удалась
		task.Status = models.StatusFailed
		errMsg := processErr.Error()
		task.ErrorMessage = &errMsg
		event = models.NewProcessingFailedEvent(sagaID, task, errMsg)
	} else {
		// Обработка успешна
		task.Status = models.StatusCompleted
		task.OutputPaths = result.OutputPaths
		task.Metadata = result.Metadata
		event = models.NewProcessingSucceededEvent(sagaID, task)
	}

	// 7. Сохраняем результат и событие в одной транзакции
	tx2, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx2: %w", err)
	}
	defer tx2.Rollback()

	if err := s.repo.UpdateTx(ctx, tx2, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	if err := s.outboxRepo.Add(ctx, tx2, string(event.Type), task.ID.String(), sagaID, event.Payload); err != nil {
		return nil, fmt.Errorf("add event to outbox: %w", err)
	}

	if err := tx2.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx2: %w", err)
	}

	return task, nil
}
