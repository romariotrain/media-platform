package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/quota/models"
	"github.com/romariotrain/media-platform/internal/quota/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ReserveQuota_Success(t *testing.T) {
	// Arrange
	repo := repository.NewMemoryRepo()
	svc := New(repo)

	userID := "user123"
	assetID := uuid.New()
	amount := int64(1024 * 1024 * 100) // 100MB

	// Устанавливаем начальную квоту пользователя
	repo.SetUserQuota(&models.UserQuota{
		UserID:      userID,
		Type:        models.StorageQuota,
		LimitAmount: 1024 * 1024 * 1024, // 1GB
		UsedAmount:  0,
		UpdatedAt:   time.Now(),
	})

	// Act
	quota, err := svc.ReserveQuota(context.Background(), userID, assetID, models.StorageQuota, amount)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, quota)
	assert.Equal(t, userID, quota.UserID)
	assert.Equal(t, assetID, quota.AssetID)
	assert.Equal(t, amount, quota.Amount)
	assert.Equal(t, models.QuotaReserved, quota.Status)
	assert.False(t, quota.ExpiresAt.IsZero())

	// Проверяем что used_amount увеличился
	userQuota, err := repo.GetUserQuota(context.Background(), userID, models.StorageQuota)
	require.NoError(t, err)
	assert.Equal(t, amount, userQuota.UsedAmount)
}

func TestService_ReserveQuota_Idempotency(t *testing.T) {
	// Arrange
	repo := repository.NewMemoryRepo()
	svc := New(repo)

	userID := "user123"
	assetID := uuid.New()
	amount := int64(1024 * 1024 * 100) // 100MB

	repo.SetUserQuota(&models.UserQuota{
		UserID:      userID,
		Type:        models.StorageQuota,
		LimitAmount: 1024 * 1024 * 1024, // 1GB
		UsedAmount:  0,
		UpdatedAt:   time.Now(),
	})

	// Act - первый раз
	quota1, err := svc.ReserveQuota(context.Background(), userID, assetID, models.StorageQuota, amount)
	require.NoError(t, err)

	// Act - второй раз с тем же assetID (идемпотентность)
	quota2, err := svc.ReserveQuota(context.Background(), userID, assetID, models.StorageQuota, amount)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, quota1.ID, quota2.ID, "должна вернуться та же квота")
	assert.Equal(t, quota1.ReservedAt, quota2.ReservedAt)

	// Проверяем что used_amount НЕ увеличился дважды
	userQuota, err := repo.GetUserQuota(context.Background(), userID, models.StorageQuota)
	require.NoError(t, err)
	assert.Equal(t, amount, userQuota.UsedAmount, "должно быть зарезервировано только один раз")
}

func TestService_ReserveQuota_InsufficientQuota(t *testing.T) {
	// Arrange
	repo := repository.NewMemoryRepo()
	svc := New(repo)

	userID := "user123"
	assetID := uuid.New()
	amount := int64(1024 * 1024 * 500) // 500MB

	// Устанавливаем квоту: лимит 1GB, уже использовано 800MB
	repo.SetUserQuota(&models.UserQuota{
		UserID:      userID,
		Type:        models.StorageQuota,
		LimitAmount: 1024 * 1024 * 1024, // 1GB
		UsedAmount:  1024 * 1024 * 800,  // 800MB использовано
		UpdatedAt:   time.Now(),
	})

	// Act - пытаемся зарезервировать 500MB (доступно только 200MB)
	quota, err := svc.ReserveQuota(context.Background(), userID, assetID, models.StorageQuota, amount)

	// Assert
	require.Error(t, err)
	assert.Nil(t, quota)
	assert.ErrorIs(t, err, models.ErrInsufficientQuota)

	// Проверяем что used_amount не изменился
	userQuota, err := repo.GetUserQuota(context.Background(), userID, models.StorageQuota)
	require.NoError(t, err)
	assert.Equal(t, int64(1024*1024*800), userQuota.UsedAmount)
}

func TestService_ReleaseQuota_Success(t *testing.T) {
	// Arrange
	repo := repository.NewMemoryRepo()
	svc := New(repo)

	userID := "user123"
	assetID := uuid.New()
	amount := int64(1024 * 1024 * 100) // 100MB

	repo.SetUserQuota(&models.UserQuota{
		UserID:      userID,
		Type:        models.StorageQuota,
		LimitAmount: 1024 * 1024 * 1024,
		UsedAmount:  0,
		UpdatedAt:   time.Now(),
	})

	// Сначала резервируем квоту
	quota, err := svc.ReserveQuota(context.Background(), userID, assetID, models.StorageQuota, amount)
	require.NoError(t, err)

	// Act - освобождаем квоту
	released, err := svc.ReleaseQuota(context.Background(), assetID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, released)
	assert.Equal(t, quota.ID, released.ID)
	assert.Equal(t, models.QuotaReleased, released.Status)
	assert.NotNil(t, released.ReleasedAt)

	// Проверяем что used_amount уменьшился обратно до 0
	userQuota, err := repo.GetUserQuota(context.Background(), userID, models.StorageQuota)
	require.NoError(t, err)
	assert.Equal(t, int64(0), userQuota.UsedAmount)
}

func TestService_ReleaseQuota_Idempotency(t *testing.T) {
	// Arrange
	repo := repository.NewMemoryRepo()
	svc := New(repo)

	userID := "user123"
	assetID := uuid.New()
	amount := int64(1024 * 1024 * 100) // 100MB

	repo.SetUserQuota(&models.UserQuota{
		UserID:      userID,
		Type:        models.StorageQuota,
		LimitAmount: 1024 * 1024 * 1024,
		UsedAmount:  0,
		UpdatedAt:   time.Now(),
	})

	// Резервируем квоту
	_, err := svc.ReserveQuota(context.Background(), userID, assetID, models.StorageQuota, amount)
	require.NoError(t, err)

	// Act - освобождаем первый раз
	released1, err := svc.ReleaseQuota(context.Background(), assetID)
	require.NoError(t, err)

	// Act - освобождаем второй раз (идемпотентность)
	released2, err := svc.ReleaseQuota(context.Background(), assetID)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, released1.ID, released2.ID)
	assert.Equal(t, models.QuotaReleased, released2.Status)

	// Проверяем что used_amount НЕ ушёл в минус
	userQuota, err := repo.GetUserQuota(context.Background(), userID, models.StorageQuota)
	require.NoError(t, err)
	assert.Equal(t, int64(0), userQuota.UsedAmount, "не должно уйти в минус")
}

func TestService_ReserveQuota_InvalidArguments(t *testing.T) {
	repo := repository.NewMemoryRepo()
	svc := New(repo)

	tests := []struct {
		name      string
		userID    string
		assetID   uuid.UUID
		amount    int64
		wantError error
	}{
		{
			name:      "empty userID",
			userID:    "",
			assetID:   uuid.New(),
			amount:    1000,
			wantError: models.ErrInvalidArgument,
		},
		{
			name:      "nil assetID",
			userID:    "user123",
			assetID:   uuid.Nil,
			amount:    1000,
			wantError: models.ErrInvalidArgument,
		},
		{
			name:      "zero amount",
			userID:    "user123",
			assetID:   uuid.New(),
			amount:    0,
			wantError: models.ErrInvalidArgument,
		},
		{
			name:      "negative amount",
			userID:    "user123",
			assetID:   uuid.New(),
			amount:    -100,
			wantError: models.ErrInvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ReserveQuota(context.Background(), tt.userID, tt.assetID, models.StorageQuota, tt.amount)
			assert.ErrorIs(t, err, tt.wantError)
		})
	}
}
