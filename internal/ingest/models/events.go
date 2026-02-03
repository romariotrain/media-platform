package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType представляет тип события
type EventType string

const (
	EventIngestReady    EventType = "IngestReady"
	EventIngestUploaded EventType = "IngestUploaded"
	EventIngestFailed   EventType = "IngestFailed"
)

// Event — базовая структура события для Kafka
type Event struct {
	MessageID string                 `json:"message_id"`
	SagaID    uuid.UUID              `json:"saga_id"`
	Type      EventType              `json:"type"`
	Step      string                 `json:"step"`
	CreatedAt time.Time              `json:"created_at"`
	Payload   map[string]interface{} `json:"payload"`
}

// NewIngestReadyEvent создаёт событие готовности к загрузке
func NewIngestReadyEvent(sagaID uuid.UUID, session *UploadSession) Event {
	return Event{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      EventIngestReady,
		Step:      "PREPARE_INGEST",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"session_id": session.ID,
			"asset_id":   session.AssetID,
			"user_id":    session.UserID,
			"token":      session.Token,
			"expires_at": session.ExpiresAt,
		},
	}
}

// NewIngestUploadedEvent создаёт событие успешной загрузки файла
func NewIngestUploadedEvent(sagaID uuid.UUID, session *UploadSession) Event {
	return Event{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      EventIngestUploaded,
		Step:      "INGEST_UPLOAD",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"session_id":   session.ID,
			"asset_id":     session.AssetID,
			"user_id":      session.UserID,
			"file_name":    session.FileName,
			"file_size":    session.FileSize,
			"content_type": session.ContentType,
			"storage_path": session.StoragePath,
		},
	}
}
