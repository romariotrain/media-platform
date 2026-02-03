package models

import (
	"time"

	"github.com/google/uuid"
)

// UploadStatus представляет статус загрузки
type UploadStatus string

const (
	StatusPending  UploadStatus = "pending"  // сессия создана, токен сгенерирован
	StatusReady    UploadStatus = "ready"    // токен готов для пользователя
	StatusUploaded UploadStatus = "uploaded" // файл получен и сохранён
	StatusExpired  UploadStatus = "expired"  // токен протух
	StatusFailed   UploadStatus = "failed"   // ошибка загрузки
)

// UploadSession представляет сессию загрузки файла
type UploadSession struct {
	ID          uuid.UUID    `db:"id"           json:"id"`
	SagaID      uuid.UUID    `db:"saga_id"      json:"saga_id"`
	AssetID     uuid.UUID    `db:"asset_id"     json:"asset_id"`
	UserID      string       `db:"user_id"      json:"user_id"`
	Token       string       `db:"token"        json:"token"`
	Status      UploadStatus `db:"status"       json:"status"`
	FileName    *string      `db:"file_name"    json:"file_name,omitempty"`
	FileSize    *int64       `db:"file_size"    json:"file_size,omitempty"`
	ContentType *string      `db:"content_type" json:"content_type,omitempty"`
	StoragePath *string      `db:"storage_path" json:"storage_path,omitempty"`
	ExpiresAt   time.Time    `db:"expires_at"   json:"expires_at"`
	CreatedAt   time.Time    `db:"created_at"   json:"created_at"`
	UpdatedAt   time.Time    `db:"updated_at"   json:"updated_at"`
}
