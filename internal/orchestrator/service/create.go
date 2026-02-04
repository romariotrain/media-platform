package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/orchestrator/models"
)

// CreateSagaRequest содержит параметры для создания новой саги
type CreateSagaRequest struct {
	AssetID uuid.UUID `json:"asset_id"`
	UserID  string    `json:"user_id"`
}

// CreateSaga создаёт новую сагу и отправляет первую команду (PrepareIngest)
func (s *Service) CreateSaga(ctx context.Context, req CreateSagaRequest) (*models.Saga, error) {
	if req.AssetID == uuid.Nil || req.UserID == "" {
		return nil, models.ErrInvalidArgument
	}

	now := s.clock()
	saga := &models.Saga{
		ID:          s.idGen(),
		AssetID:     req.AssetID,
		UserID:      req.UserID,
		Status:      models.SagaStatusRunning,
		CurrentStep: models.StepPrepareIngest,
		Payload:     &models.SagaPayload{},
		StartedAt:   &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Создаём команду для Ingest сервиса
	cmd := models.NewPrepareIngestCommand(saga.ID, saga.AssetID, saga.UserID)

	// Сохраняем сагу и команду в одной транзакции
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.CreateTx(ctx, tx, saga); err != nil {
		return nil, fmt.Errorf("create saga: %w", err)
	}

	if err := s.outboxRepo.Add(ctx, tx, string(cmd.Type), saga.ID.String(), saga.ID, cmd); err != nil {
		return nil, fmt.Errorf("add command to outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return saga, nil
}

// GetByID возвращает сагу по ID
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*models.Saga, error) {
	if id == uuid.Nil {
		return nil, models.ErrInvalidArgument
	}
	return s.repo.GetByID(ctx, id)
}

// ListByUserID возвращает список саг пользователя
func (s *Service) ListByUserID(ctx context.Context, userID string) ([]*models.Saga, error) {
	if userID == "" {
		return nil, models.ErrInvalidArgument
	}
	return s.repo.ListByUserID(ctx, userID)
}
