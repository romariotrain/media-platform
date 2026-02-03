package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType представляет тип события
type EventType string

const (
	EventProcessingSucceeded EventType = "ProcessingSucceeded"
	EventProcessingFailed    EventType = "ProcessingFailed"
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

// NewProcessingSucceededEvent создаёт событие успешной обработки
func NewProcessingSucceededEvent(sagaID uuid.UUID, task *ProcessingTask) Event {
	payload := map[string]interface{}{
		"task_id":  task.ID,
		"asset_id": task.AssetID,
	}

	if task.OutputPaths != nil {
		payload["output_paths"] = task.OutputPaths
	}
	if task.Metadata != nil {
		payload["metadata"] = task.Metadata
	}

	return Event{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      EventProcessingSucceeded,
		Step:      "PROCESS_MEDIA",
		CreatedAt: time.Now(),
		Payload:   payload,
	}
}

// NewProcessingFailedEvent создаёт событие неудачной обработки
func NewProcessingFailedEvent(sagaID uuid.UUID, task *ProcessingTask, errorMessage string) Event {
	return Event{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      EventProcessingFailed,
		Step:      "PROCESS_MEDIA",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"task_id":       task.ID,
			"asset_id":      task.AssetID,
			"error_message": errorMessage,
		},
	}
}
