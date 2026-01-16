package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/romariotrain/media-platform/internal/media/models"
)

type StoreMock struct {
	mock.Mock
}

func (m *StoreMock) Create(ctx context.Context, media *models.Media) error {
	args := m.Called(ctx, media)
	return args.Error(0)
}

func (m *StoreMock) GetByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	args := m.Called(ctx, id)
	if v := args.Get(0); v != nil {
		return v.(*models.Media), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *StoreMock) UpdateStatus(ctx context.Context, id uuid.UUID, status models.Status) (*models.Media, error) {
	args := m.Called(ctx, id, status)
	if v := args.Get(0); v != nil {
		return v.(*models.Media), args.Error(1)
	}
	return nil, args.Error(1)
}
