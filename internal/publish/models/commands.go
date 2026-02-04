package models

import "github.com/google/uuid"

// CommandType представляет тип команды
type CommandType string

const (
	CommandStartPublish CommandType = "StartPublish"
)

// Command — envelope для Kafka-команд от оркестратора
type Command struct {
	MessageID string                 `json:"message_id"`
	SagaID    uuid.UUID              `json:"saga_id"`
	Type      CommandType            `json:"type"`
	Step      string                 `json:"step"`
	Payload   map[string]interface{} `json:"payload"`
}

// StartPublishPayload — payload команды StartPublish
type StartPublishPayload struct {
	AssetID    uuid.UUID    `json:"asset_id"`
	SourcePaths *SourcePaths `json:"source_paths"`
}
