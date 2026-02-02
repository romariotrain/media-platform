package models

import (
	"time"

	"github.com/google/uuid"
)

// EventType представляет тип события
type EventType string

const (
	EventQuotaReserved EventType = "QuotaReserved"
	EventQuotaReleased EventType = "QuotaReleased"
	EventQuotaFailed   EventType = "QuotaFailed"
)

// Event базовая структура события для Kafka
type Event struct {
	MessageID string                 `json:"message_id"` // для идемпотентности
	SagaID    uuid.UUID              `json:"saga_id"`    // корреляция с сагой
	Type      EventType              `json:"type"`       // тип события
	Step      string                 `json:"step"`       // шаг саги (для логов)
	CreatedAt time.Time              `json:"created_at"`
	Payload   map[string]interface{} `json:"payload"`
}

// QuotaReservedPayload данные успешной резервации
type QuotaReservedPayload struct {
	QuotaID   uuid.UUID `json:"quota_id"`
	AssetID   uuid.UUID `json:"asset_id"`
	UserID    string    `json:"user_id"`
	Type      QuotaType `json:"type"`
	Amount    int64     `json:"amount"`
	ExpiresAt time.Time `json:"expires_at"`
}

// QuotaReleasedPayload данные освобождения квоты
type QuotaReleasedPayload struct {
	QuotaID    uuid.UUID `json:"quota_id"`
	AssetID    uuid.UUID `json:"asset_id"`
	UserID     string    `json:"user_id"`
	Amount     int64     `json:"amount"`
	ReleasedAt time.Time `json:"released_at"`
}

// QuotaFailedPayload данные неудачной резервации
type QuotaFailedPayload struct {
	AssetID   uuid.UUID `json:"asset_id"`
	UserID    string    `json:"user_id"`
	Type      QuotaType `json:"type"`
	Requested int64     `json:"requested"` // сколько запросили
	Available int64     `json:"available"` // сколько доступно
	Reason    string    `json:"reason"`    // причина отказа
}

// NewQuotaReservedEvent создаёт событие успешной резервации
func NewQuotaReservedEvent(sagaID uuid.UUID, quota *Quota) Event {
	return Event{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      EventQuotaReserved,
		Step:      "RESERVE_QUOTA",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"quota_id":   quota.ID,
			"asset_id":   quota.AssetID,
			"user_id":    quota.UserID,
			"type":       quota.Type,
			"amount":     quota.Amount,
			"expires_at": quota.ExpiresAt,
		},
	}
}

// NewQuotaReleasedEvent создаёт событие освобождения квоты
func NewQuotaReleasedEvent(sagaID uuid.UUID, quota *Quota) Event {
	return Event{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      EventQuotaReleased,
		Step:      "RELEASE_QUOTA",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"quota_id":    quota.ID,
			"asset_id":    quota.AssetID,
			"user_id":     quota.UserID,
			"amount":      quota.Amount,
			"released_at": time.Now(),
		},
	}
}

// NewQuotaFailedEvent создаёт событие неудачной резервации
func NewQuotaFailedEvent(sagaID, assetID uuid.UUID, userID string, quotaType QuotaType, requested, available int64, reason string) Event {
	return Event{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      EventQuotaFailed,
		Step:      "RESERVE_QUOTA",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"asset_id":  assetID,
			"user_id":   userID,
			"type":      quotaType,
			"requested": requested,
			"available": available,
			"reason":    reason,
		},
	}
}
