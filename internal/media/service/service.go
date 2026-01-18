package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/media/domain"
	"github.com/romariotrain/media-platform/internal/storage/postgres"

	"github.com/romariotrain/media-platform/internal/media/models"
	"github.com/romariotrain/media-platform/internal/media/repository"
)

type Service struct {
	repo       repository.MediaRepository
	clock      func() time.Time
	idGen      func() uuid.UUID
	outboxRepo *postgres.OutboxRepo
}

func New(repo repository.MediaRepository, outboxRepo *postgres.OutboxRepo) *Service {
	return &Service{
		repo:       repo,
		outboxRepo: outboxRepo, // добавь это
		clock:      time.Now,
		idGen:      uuid.New,
	}
}

// GetMedia returns Media by id. It simply delegates to repository and passes through
// domain errors (e.g. models.ErrNotFound) so the transport layer can map them to HTTP.
func (s *Service) GetMedia(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	if id == uuid.Nil {
		return nil, models.ErrInvalidArgument
	}
	return s.repo.GetByID(ctx, id)
}

// CreateMedia creates a new Media entity and persists it via repository.
// Service owns invariants: id, initial status, timestamps, basic validation.
func (s *Service) CreateMedia(ctx context.Context, mediaType models.MediaType, source string) (*models.Media, error) {
	if mediaType == "" || source == "" {
		return nil, models.ErrInvalidArgument
	}

	now := s.clock()

	m := &models.Media{
		ID:        s.idGen(),
		Status:    models.UploadedStatus,
		Type:      mediaType,
		Source:    source,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.repo.Create(ctx, m); err != nil {
		return nil, err
	}

	return m, nil
}

func toDomainStatus(s models.Status) (domain.Status, error) {
	switch s {
	case models.UploadedStatus:
		return domain.Uploaded, nil
	case models.ProcessingStatus:
		return domain.Processing, nil
	case models.ReadyStatus:
		return domain.Ready, nil
	case models.FailedStatus:
		return domain.Failed, nil
	default:
		return "", fmt.Errorf("unknown status: %s", s)
	}
}

func (s *Service) ChangeStatus(ctx context.Context, id uuid.UUID, to models.Status) (*models.Media, error) {
	// 1. Получаем текущую медиа (чтобы узнать старый статус)
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 2. Валидация перехода (твоя логика)
	fromDom, err := toDomainStatus(m.Status)
	if err != nil {
		return nil, err
	}
	toDom, err := toDomainStatus(to)
	if err != nil {
		return nil, err
	}
	if err := domain.ValidateTransition(fromDom, toDom); err != nil {
		return nil, err
	}

	// Если статус уже такой — ничего не делаем
	if m.Status == to {
		return m, nil
	}

	// 3. НАЧИНАЕМ ТРАНЗАКЦИЮ
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback() // откатится если не сделаем Commit

	// 4. Обновляем статус (В ТРАНЗАКЦИИ)
	updated, err := s.repo.UpdateStatusTx(ctx, tx, id, to)
	if err != nil {
		return nil, err
	}

	// 5. Создаём событие
	event := models.NewMediaStatusChanged(id, m.Status, to)

	// 6. Добавляем в outbox (В ТОЙ ЖЕ ТРАНЗАКЦИИ)
	if err := s.outboxRepo.Add(ctx, tx, event); err != nil {
		return nil, fmt.Errorf("add outbox: %w", err)
	}

	// 7. КОММИТИМ (атомарно!)
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return updated, nil
}
