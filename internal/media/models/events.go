package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface {
	EventID() uuid.UUID
	EventType() string
	AggregateID() uuid.UUID
	OccurredAt() time.Time
}

type MediaStatusChanged struct {
	eventID    uuid.UUID
	mediaID    uuid.UUID
	from       Status
	to         Status
	occurredAt time.Time
}

func NewMediaStatusChanged(mediaID uuid.UUID, from, to Status) *MediaStatusChanged {
	return &MediaStatusChanged{
		eventID:    uuid.New(),
		mediaID:    mediaID,
		from:       from,
		to:         to,
		occurredAt: time.Now(),
	}
}

// Реализация интерфейса DomainEvent
func (e *MediaStatusChanged) EventID() uuid.UUID     { return e.eventID }
func (e *MediaStatusChanged) EventType() string      { return "MediaStatusChanged" }
func (e *MediaStatusChanged) AggregateID() uuid.UUID { return e.mediaID }
func (e *MediaStatusChanged) OccurredAt() time.Time  { return e.occurredAt }

// Геттеры для payload
func (e *MediaStatusChanged) From() Status { return e.from }
func (e *MediaStatusChanged) To() Status   { return e.to }

// Кастомная JSON сериализация
func (e *MediaStatusChanged) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		EventID    uuid.UUID `json:"event_id"`
		MediaID    uuid.UUID `json:"media_id"`
		From       Status    `json:"from"`
		To         Status    `json:"to"`
		OccurredAt time.Time `json:"occurred_at"`
	}{
		EventID:    e.eventID,
		MediaID:    e.mediaID,
		From:       e.from,
		To:         e.to,
		OccurredAt: e.occurredAt,
	})
}
