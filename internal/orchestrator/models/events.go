package models

import "github.com/google/uuid"

// IncomingEventType — типы событий, приходящих от других сервисов
type IncomingEventType string

const (
	EventIngestReady          IncomingEventType = "IngestReady"
	EventIngestUploaded       IncomingEventType = "IngestUploaded"
	EventProcessingSucceeded  IncomingEventType = "ProcessingSucceeded"
	EventProcessingFailed     IncomingEventType = "ProcessingFailed"
	EventPublishSucceeded     IncomingEventType = "PublishSucceeded"
	EventPublishFailed        IncomingEventType = "PublishFailed"
)

// IncomingEvent — событие от другого сервиса
type IncomingEvent struct {
	MessageID string                 `json:"message_id"`
	SagaID    uuid.UUID              `json:"saga_id"`
	Type      IncomingEventType      `json:"type"`
	Step      string                 `json:"step"`
	Payload   map[string]interface{} `json:"payload"`
}
