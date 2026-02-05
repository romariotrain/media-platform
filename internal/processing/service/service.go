package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/processing/ffmpeg"
	"github.com/romariotrain/media-platform/internal/processing/models"
	"github.com/romariotrain/media-platform/internal/processing/repository"
	"github.com/romariotrain/media-platform/internal/processing/storage/postgres"
)

// Processor интерфейс для обработки медиафайлов
type Processor interface {
	Process(ctx context.Context, inputPath string, taskID string) (*ffmpeg.ProcessResult, error)
}

// Service реализует бизнес-логику Processing сервиса
type Service struct {
	repo       repository.ProcessingTaskRepository
	outboxRepo *postgres.OutboxRepo
	processor  Processor
	clock      func() time.Time
	idGen      func() uuid.UUID
}

// New создаёт новый экземпляр Service
func New(
	repo repository.ProcessingTaskRepository,
	outboxRepo *postgres.OutboxRepo,
	processor Processor,
) *Service {
	return &Service{
		repo:       repo,
		outboxRepo: outboxRepo,
		processor:  processor,
		clock:      time.Now,
		idGen:      uuid.New,
	}
}

// MockProcessor — простой процессор, который копирует файл без обработки
type MockProcessor struct {
	outputDir string
}

// NewMockProcessor создаёт mock-процессор
func NewMockProcessor(outputDir string) *MockProcessor {
	return &MockProcessor{outputDir: outputDir}
}

// Process копирует файл в output директорию (без реальной обработки)
func (m *MockProcessor) Process(ctx context.Context, inputPath string, taskID string) (*ffmpeg.ProcessResult, error) {
	return &ffmpeg.ProcessResult{
		OutputPaths: &models.OutputPaths{
			Video720p: inputPath,
			Thumbnail: inputPath,
		},
		Metadata: &models.Metadata{
			Width:    1280,
			Height:   720,
			Duration: 0,
			Format:   "mock",
		},
	}, nil
}
