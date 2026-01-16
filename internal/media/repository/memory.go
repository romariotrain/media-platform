package repository

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/romariotrain/media-platform/internal/media/models"
)

type MemoryRepository struct {
	mu   sync.RWMutex
	data map[uuid.UUID]*models.Media
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		data: make(map[uuid.UUID]*models.Media),
	}
}

func (r *MemoryRepository) Create(ctx context.Context, m *models.Media) error {
	if m == nil {
		return models.ErrInvalidArgument
	}
	if m.ID == uuid.Nil {
		return models.ErrInvalidArgument
	}

	//TODO ctx на будущее (таймауты/кансел), для in-memory просто проверим отмену
	if err := ctx.Err(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[m.ID]; exists {
		return models.ErrConflict
	}

	// Защитная копия, чтобы внешняя сторона не могла мутировать хранимый объект
	cp := *m
	r.data[m.ID] = &cp

	return nil
}

func (r *MemoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	if id == uuid.Nil {
		return nil, models.ErrInvalidArgument
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.data[id]
	if !ok {
		return nil, models.ErrNotFound
	}

	// Возвращаем копию, чтобы не было внешних мутаций
	cp := *m
	return &cp, nil
}
