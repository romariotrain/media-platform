package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/ingest/repository"
	"github.com/romariotrain/media-platform/internal/ingest/storage/fs"
	"github.com/romariotrain/media-platform/internal/ingest/storage/postgres"
)

const (
	// UploadTokenTTL — время жизни токена загрузки
	UploadTokenTTL = 30 * time.Minute

	// MaxFileSize — максимальный размер файла (500MB)
	MaxFileSize = 1024 * 1024 * 500
)

// AllowedContentTypes — допустимые MIME-типы для загрузки
var AllowedContentTypes = map[string]bool{
	"video/mp4":       true,
	"video/webm":      true,
	"audio/mpeg":      true,
	"audio/wav":       true,
	"image/jpeg":      true,
	"image/png":       true,
	"application/pdf": true,
}

// Service реализует бизнес-логику Ingest сервиса
type Service struct {
	repo       repository.UploadSessionRepository
	outboxRepo *postgres.OutboxRepo
	fileStore  *fs.FileStore
	clock      func() time.Time
	idGen      func() uuid.UUID
	tokenGen   func() string
}

// New создаёт новый экземпляр Service
func New(
	repo repository.UploadSessionRepository,
	outboxRepo *postgres.OutboxRepo,
	fileStore *fs.FileStore,
) *Service {
	return &Service{
		repo:       repo,
		outboxRepo: outboxRepo,
		fileStore:  fileStore,
		clock:      time.Now,
		idGen:      uuid.New,
		tokenGen:   func() string { return uuid.New().String() },
	}
}
