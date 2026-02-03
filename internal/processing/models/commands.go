package models

import "github.com/google/uuid"

// CommandType представляет тип команды
type CommandType string

const (
	CommandStartProcessing CommandType = "StartProcessing"
)

// Command — envelope для Kafka-команд от оркестратора
type Command struct {
	MessageID string                 `json:"message_id"`
	SagaID    uuid.UUID              `json:"saga_id"`
	Type      CommandType            `json:"type"`
	Step      string                 `json:"step"`
	Payload   map[string]interface{} `json:"payload"`
}

// StartProcessingPayload — payload команды StartProcessing
type StartProcessingPayload struct {
	AssetID   uuid.UUID `json:"asset_id"`
	InputPath string    `json:"input_path"`
}
