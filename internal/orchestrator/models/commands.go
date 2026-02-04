package models

import (
	"time"

	"github.com/google/uuid"
)

// OutgoingCommandType — типы команд, отправляемых в другие сервисы
type OutgoingCommandType string

const (
	CmdPrepareIngest    OutgoingCommandType = "PrepareIngest"
	CmdStartProcessing  OutgoingCommandType = "StartProcessing"
	CmdStartPublish     OutgoingCommandType = "StartPublish"
)

// OutgoingCommand — команда для отправки в другой сервис через outbox
type OutgoingCommand struct {
	MessageID string                 `json:"message_id"`
	SagaID    uuid.UUID              `json:"saga_id"`
	Type      OutgoingCommandType    `json:"type"`
	Step      string                 `json:"step"`
	CreatedAt time.Time              `json:"created_at"`
	Payload   map[string]interface{} `json:"payload"`
}

// NewPrepareIngestCommand создаёт команду для Ingest сервиса
func NewPrepareIngestCommand(sagaID, assetID uuid.UUID, userID string) OutgoingCommand {
	return OutgoingCommand{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      CmdPrepareIngest,
		Step:      "PREPARE_INGEST",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"asset_id": assetID,
			"user_id":  userID,
		},
	}
}

// NewStartProcessingCommand создаёт команду для Processing сервиса
func NewStartProcessingCommand(sagaID, assetID uuid.UUID, inputPath string) OutgoingCommand {
	return OutgoingCommand{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      CmdStartProcessing,
		Step:      "PROCESS_MEDIA",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"asset_id":   assetID,
			"input_path": inputPath,
		},
	}
}

// NewStartPublishCommand создаёт команду для Publish сервиса
func NewStartPublishCommand(sagaID, assetID uuid.UUID, outputPaths interface{}) OutgoingCommand {
	return OutgoingCommand{
		MessageID: uuid.New().String(),
		SagaID:    sagaID,
		Type:      CmdStartPublish,
		Step:      "PUBLISH_MEDIA",
		CreatedAt: time.Now(),
		Payload: map[string]interface{}{
			"asset_id":     assetID,
			"source_paths": outputPaths,
		},
	}
}
