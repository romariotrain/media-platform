package models

import "github.com/google/uuid"

// CommandType представляет тип команды
type CommandType string

const (
	CommandPrepareIngest CommandType = "PrepareIngest"
)

// Command — envelope для Kafka-команд от оркестратора
type Command struct {
	MessageID string                 `json:"message_id"`
	SagaID    uuid.UUID              `json:"saga_id"`
	Type      CommandType            `json:"type"`
	Step      string                 `json:"step"`
	Payload   map[string]interface{} `json:"payload"`
}

// PrepareIngestPayload — payload команды PrepareIngest
type PrepareIngestPayload struct {
	AssetID uuid.UUID `json:"asset_id"`
	UserID  string    `json:"user_id"`
}
