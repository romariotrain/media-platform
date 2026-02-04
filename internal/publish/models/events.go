package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType представляет тип события
type EventType string

const (
	EventPublishSucceeded EventType = "PublishSucceeded"
	EventPublishFailed    EventType = "PublishFailed"
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

// NewPublishSucceededEvent создаёт событие успешной публикации
func NewPublishSucceededEvent(sagaID uuid.UUID, pub *Publication) Event {
	payload := map[string]interface{}{
		"publication_id": pub.ID,
		"asset_id":       pub.AssetID,
	}

	if pub.PublicURLs != nil {
		payload["public_urls"] = pub.PublicURLs
	}

	return Event{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      EventPublishSucceeded,
		Step:      "PUBLISH_MEDIA",
		CreatedAt: time.Now(),
		Payload:   payload,
	}
}

// NewPublishFailedEvent создаёт событие неудачной публикации
func NewPublishFailedEvent(sagaID uuid.UUID, pub *Publication, errorMessage string) Event {
	return Event{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      EventPublishFailed,
		Step:      "PUBLISH_MEDIA",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"publication_id": pub.ID,
			"asset_id":       pub.AssetID,
			"error_message":  errorMessage,
		},
	}
}
