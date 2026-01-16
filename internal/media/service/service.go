package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/media/domain"

	"github.com/romariotrain/media-platform/internal/media/models"
	"github.com/romariotrain/media-platform/internal/media/repository"
)

type Service struct {
	repo  repository.MediaRepository
	clock func() time.Time
	idGen func() uuid.UUID
}

func New(repo repository.MediaRepository) *Service {
	return &Service{
		repo:  repo,
		clock: time.Now,
		idGen: uuid.New,
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
	m, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

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

	if m.Status == to {
		return m, nil
	}

	updated, err := s.repo.UpdateStatus(ctx, id, to)
	if err != nil {
		return nil, err
	}

	return updated, nil
}
