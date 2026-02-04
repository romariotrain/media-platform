package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/orchestrator/repository"
	"github.com/romariotrain/media-platform/internal/orchestrator/storage/postgres"
)

// Service реализует бизнес-логику Orchestrator сервиса
type Service struct {
	repo       repository.SagaRepository
	outboxRepo *postgres.OutboxRepo
	clock      func() time.Time
	idGen      func() uuid.UUID
}

// New создаёт новый экземпляр Service
func New(
	repo repository.SagaRepository,
	outboxRepo *postgres.OutboxRepo,
) *Service {
	return &Service{
		repo:       repo,
		outboxRepo: outboxRepo,
		clock:      time.Now,
		idGen:      uuid.New,
	}
}
