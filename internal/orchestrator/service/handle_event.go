package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/orchestrator/models"
	"github.com/rs/zerolog/log"
)

// HandleEvent обрабатывает входящее событие от другого сервиса и продвигает сагу
func (s *Service) HandleEvent(ctx context.Context, event models.IncomingEvent) error {
	if event.SagaID == uuid.Nil {
		return models.ErrInvalidArgument
	}

	saga, err := s.repo.GetByID(ctx, event.SagaID)
	if err != nil {
		return fmt.Errorf("get saga: %w", err)
	}

	if saga.IsTerminal() {
		log.Warn().
			Str("saga_id", saga.ID.String()).
			Str("status", string(saga.Status)).
			Str("event_type", string(event.Type)).
			Msg("received event for terminal saga, ignoring")
		return nil
	}

	switch event.Type {
	case models.EventIngestReady:
		return s.handleIngestReady(ctx, saga, event)
	case models.EventIngestUploaded:
		return s.handleIngestUploaded(ctx, saga, event)
	case models.EventProcessingSucceeded:
		return s.handleProcessingSucceeded(ctx, saga, event)
	case models.EventProcessingFailed:
		return s.handleStepFailed(ctx, saga, event)
	case models.EventPublishSucceeded:
		return s.handlePublishSucceeded(ctx, saga, event)
	case models.EventPublishFailed:
		return s.handleStepFailed(ctx, saga, event)
	default:
		log.Warn().
			Str("saga_id", saga.ID.String()).
			Str("event_type", string(event.Type)).
			Msg("unknown event type")
		return nil
	}
}

// handleIngestReady — Ingest подготовил токен, ждём загрузки файла
func (s *Service) handleIngestReady(ctx context.Context, saga *models.Saga, event models.IncomingEvent) error {
	if saga.CurrentStep != models.StepPrepareIngest {
		log.Warn().
			Str("saga_id", saga.ID.String()).
			Str("current_step", string(saga.CurrentStep)).
			Msg("unexpected IngestReady event for current step")
		return nil
	}

	saga.CurrentStep = models.StepWaitUpload
	saga.UpdatedAt = s.clock()

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.UpdateTx(ctx, tx, saga); err != nil {
		return fmt.Errorf("update saga: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	log.Info().
		Str("saga_id", saga.ID.String()).
		Msg("saga advanced to WAIT_UPLOAD")

	return nil
}

// handleIngestUploaded — файл загружен, запускаем обработку
func (s *Service) handleIngestUploaded(ctx context.Context, saga *models.Saga, event models.IncomingEvent) error {
	if saga.CurrentStep != models.StepWaitUpload {
		log.Warn().
			Str("saga_id", saga.ID.String()).
			Str("current_step", string(saga.CurrentStep)).
			Msg("unexpected IngestUploaded event for current step")
		return nil
	}

	// Извлекаем storage_path из payload
	storagePath, _ := event.Payload["storage_path"].(string)
	if storagePath == "" {
		return fmt.Errorf("missing storage_path in IngestUploaded payload")
	}

	// Обновляем payload саги
	if saga.Payload == nil {
		saga.Payload = &models.SagaPayload{}
	}
	saga.Payload.StoragePath = storagePath
	if fn, ok := event.Payload["file_name"].(string); ok {
		saga.Payload.FileName = fn
	}
	if ct, ok := event.Payload["content_type"].(string); ok {
		saga.Payload.ContentType = ct
	}

	saga.CurrentStep = models.StepProcessMedia
	saga.UpdatedAt = s.clock()

	// Создаём команду для Processing
	cmd := models.NewStartProcessingCommand(saga.ID, saga.AssetID, storagePath)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.UpdateTx(ctx, tx, saga); err != nil {
		return fmt.Errorf("update saga: %w", err)
	}

	if err := s.outboxRepo.Add(ctx, tx, string(cmd.Type), saga.ID.String(), saga.ID, cmd); err != nil {
		return fmt.Errorf("add command to outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	log.Info().
		Str("saga_id", saga.ID.String()).
		Str("storage_path", storagePath).
		Msg("saga advanced to PROCESS_MEDIA")

	return nil
}

// handleProcessingSucceeded — обработка завершена, запускаем публикацию
func (s *Service) handleProcessingSucceeded(ctx context.Context, saga *models.Saga, event models.IncomingEvent) error {
	if saga.CurrentStep != models.StepProcessMedia {
		log.Warn().
			Str("saga_id", saga.ID.String()).
			Str("current_step", string(saga.CurrentStep)).
			Msg("unexpected ProcessingSucceeded event for current step")
		return nil
	}

	// Извлекаем output_paths и metadata из payload
	outputPaths := event.Payload["output_paths"]
	metadata := event.Payload["metadata"]

	if saga.Payload == nil {
		saga.Payload = &models.SagaPayload{}
	}
	saga.Payload.OutputPaths = outputPaths
	saga.Payload.Metadata = metadata

	saga.CurrentStep = models.StepPublishMedia
	saga.UpdatedAt = s.clock()

	// Создаём команду для Publish
	cmd := models.NewStartPublishCommand(saga.ID, saga.AssetID, outputPaths)

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.UpdateTx(ctx, tx, saga); err != nil {
		return fmt.Errorf("update saga: %w", err)
	}

	if err := s.outboxRepo.Add(ctx, tx, string(cmd.Type), saga.ID.String(), saga.ID, cmd); err != nil {
		return fmt.Errorf("add command to outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	log.Info().
		Str("saga_id", saga.ID.String()).
		Msg("saga advanced to PUBLISH_MEDIA")

	return nil
}

// handlePublishSucceeded — публикация завершена, сага закончена
func (s *Service) handlePublishSucceeded(ctx context.Context, saga *models.Saga, event models.IncomingEvent) error {
	if saga.CurrentStep != models.StepPublishMedia {
		log.Warn().
			Str("saga_id", saga.ID.String()).
			Str("current_step", string(saga.CurrentStep)).
			Msg("unexpected PublishSucceeded event for current step")
		return nil
	}

	if saga.Payload == nil {
		saga.Payload = &models.SagaPayload{}
	}
	saga.Payload.PublicURLs = event.Payload["public_urls"]

	now := s.clock()
	saga.CurrentStep = models.StepDone
	saga.Status = models.SagaStatusCompleted
	saga.CompletedAt = &now
	saga.UpdatedAt = now

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.UpdateTx(ctx, tx, saga); err != nil {
		return fmt.Errorf("update saga: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	log.Info().
		Str("saga_id", saga.ID.String()).
		Msg("saga completed successfully")

	return nil
}

// handleStepFailed — шаг завершился с ошибкой, помечаем сагу как failed
func (s *Service) handleStepFailed(ctx context.Context, saga *models.Saga, event models.IncomingEvent) error {
	errMsg := "unknown error"
	if msg, ok := event.Payload["error_message"].(string); ok {
		errMsg = msg
	}

	now := s.clock()
	saga.Status = models.SagaStatusFailed
	saga.ErrorMessage = &errMsg
	saga.CompletedAt = &now
	saga.UpdatedAt = now

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.UpdateTx(ctx, tx, saga); err != nil {
		return fmt.Errorf("update saga: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	log.Error().
		Str("saga_id", saga.ID.String()).
		Str("step", string(saga.CurrentStep)).
		Str("error", errMsg).
		Msg("saga failed")

	return nil
}
