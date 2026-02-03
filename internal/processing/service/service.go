package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/processing/ffmpeg"
	"github.com/romariotrain/media-platform/internal/processing/repository"
	"github.com/romariotrain/media-platform/internal/processing/storage/postgres"
)

// Service реализует бизнес-логику Processing сервиса
type Service struct {
	repo       repository.ProcessingTaskRepository
	outboxRepo *postgres.OutboxRepo
	processor  *ffmpeg.Processor
	clock      func() time.Time
	idGen      func() uuid.UUID
}

// New создаёт новый экземпляр Service
func New(
	repo repository.ProcessingTaskRepository,
	outboxRepo *postgres.OutboxRepo,
	processor *ffmpeg.Processor,
) *Service {
	return &Service{
		repo:       repo,
		outboxRepo: outboxRepo,
		processor:  processor,
		clock:      time.Now,
		idGen:      uuid.New,
	}
}
