package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/publish/repository"
	"github.com/romariotrain/media-platform/internal/publish/storage/fs"
	"github.com/romariotrain/media-platform/internal/publish/storage/postgres"
)

// Service реализует бизнес-логику Publish сервиса
type Service struct {
	repo       repository.PublicationRepository
	outboxRepo *postgres.OutboxRepo
	publisher  *fs.FilePublisher
	clock      func() time.Time
	idGen      func() uuid.UUID
}

// New создаёт новый экземпляр Service
func New(
	repo repository.PublicationRepository,
	outboxRepo *postgres.OutboxRepo,
	publisher *fs.FilePublisher,
) *Service {
	return &Service{
		repo:       repo,
		outboxRepo: outboxRepo,
		publisher:  publisher,
		clock:      time.Now,
		idGen:      uuid.New,
	}
}
