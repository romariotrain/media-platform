package httpapi

import (
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/ingest/models"
)

// UploadTokenResponse — ответ с информацией о токене загрузки
type UploadTokenResponse struct {
	SessionID uuid.UUID `json:"session_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    string    `json:"status"`
}

// UploadResponse — ответ после успешной загрузки файла
type UploadResponse struct {
	SessionID   uuid.UUID `json:"session_id"`
	AssetID     uuid.UUID `json:"asset_id"`
	Status      string    `json:"status"`
	FileName    string    `json:"file_name"`
	FileSize    int64     `json:"file_size"`
	ContentType string    `json:"content_type"`
	StoragePath string    `json:"storage_path"`
}

func toUploadTokenResponse(s *models.UploadSession) UploadTokenResponse {
	return UploadTokenResponse{
		SessionID: s.ID,
		Token:     s.Token,
		ExpiresAt: s.ExpiresAt,
		Status:    string(s.Status),
	}
}

func toUploadResponse(s *models.UploadSession) UploadResponse {
	resp := UploadResponse{
		SessionID: s.ID,
		AssetID:   s.AssetID,
		Status:    string(s.Status),
	}
	if s.FileName != nil {
		resp.FileName = *s.FileName
	}
	if s.FileSize != nil {
		resp.FileSize = *s.FileSize
	}
	if s.ContentType != nil {
		resp.ContentType = *s.ContentType
	}
	if s.StoragePath != nil {
		resp.StoragePath = *s.StoragePath
	}
	return resp
}
