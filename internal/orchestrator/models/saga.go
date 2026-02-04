package models

import (
	"time"

	"github.com/google/uuid"
)

// SagaStatus представляет общий статус саги
type SagaStatus string

const (
	SagaStatusCreated    SagaStatus = "created"
	SagaStatusRunning    SagaStatus = "running"
	SagaStatusCompleted  SagaStatus = "completed"
	SagaStatusFailed     SagaStatus = "failed"
)

// SagaStep представляет текущий шаг саги
type SagaStep string

const (
	StepPrepareIngest SagaStep = "PREPARE_INGEST"
	StepWaitUpload    SagaStep = "WAIT_UPLOAD"
	StepProcessMedia  SagaStep = "PROCESS_MEDIA"
	StepPublishMedia  SagaStep = "PUBLISH_MEDIA"
	StepDone          SagaStep = "DONE"
)

// SagaPayload содержит данные, передаваемые между шагами
type SagaPayload struct {
	FileName    string      `json:"file_name,omitempty"`
	ContentType string      `json:"content_type,omitempty"`
	StoragePath string      `json:"storage_path,omitempty"`
	OutputPaths interface{} `json:"output_paths,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
	PublicURLs  interface{} `json:"public_urls,omitempty"`
}

// Saga представляет запись оркестрации пайплайна
type Saga struct {
	ID           uuid.UUID    `db:"id"              json:"id"`
	AssetID      uuid.UUID    `db:"asset_id"        json:"asset_id"`
	UserID       string       `db:"user_id"         json:"user_id"`
	Status       SagaStatus   `db:"status"          json:"status"`
	CurrentStep  SagaStep     `db:"current_step"    json:"current_step"`
	Payload      *SagaPayload `db:"payload"          json:"payload,omitempty"`
	ErrorMessage *string      `db:"error_message"   json:"error_message,omitempty"`
	StartedAt    *time.Time   `db:"started_at"      json:"started_at,omitempty"`
	CompletedAt  *time.Time   `db:"completed_at"    json:"completed_at,omitempty"`
	CreatedAt    time.Time    `db:"created_at"      json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at"      json:"updated_at"`
}

// IsTerminal возвращает true если сага в конечном состоянии
func (s *Saga) IsTerminal() bool {
	return s.Status == SagaStatusCompleted || s.Status == SagaStatusFailed
}

// NextStep возвращает следующий шаг после успешного завершения текущего
func NextStep(current SagaStep) (SagaStep, bool) {
	switch current {
	case StepPrepareIngest:
		return StepWaitUpload, true
	case StepWaitUpload:
		return StepProcessMedia, true
	case StepProcessMedia:
		return StepPublishMedia, true
	case StepPublishMedia:
		return StepDone, true
	default:
		return "", false
	}
}
