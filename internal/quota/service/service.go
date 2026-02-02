package service

import (
	"context"
	_ "database/sql"
	"time"

	"github.com/google/uuid"
	_ "github.com/jmoiron/sqlx"
	_ "github.com/romariotrain/media-platform/internal/quota/models"
	"github.com/romariotrain/media-platform/internal/quota/repository"
)

// OutboxRepo интерфейс для работы с outbox
type OutboxRepo interface {
	Add(ctx context.Context, tx repository.Tx, eventType string, sagaID uuid.UUID, payload interface{}) error
}

// Service содержит бизнес-логику работы с квотами
type Service struct {
	repo       repository.QuotaRepository
	outboxRepo OutboxRepo
	clock      func() time.Time
	idGen      func() uuid.UUID
}

// New создаёт новый экземпляр Service
func New(repo repository.QuotaRepository, outboxRepo OutboxRepo) *Service {
	return &Service{
		repo:       repo,
		outboxRepo: outboxRepo,
		clock:      time.Now,
		idGen:      uuid.New,
	}
}

// QuotaReservationTTL время жизни резервации (1 час)
const QuotaReservationTTL = 1 * time.Hour
