package models

import (
	"time"

	"github.com/google/uuid"
)

// JobStatus представляет статус задачи обработки
type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusProcessing JobStatus = "processing"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
)

// ProcessingJob представляет задачу обработки медиафайла
type ProcessingJob struct {
	ID           uuid.UUID  `db:"id" json:"id"`
	SagaID       uuid.UUID  `db:"saga_id" json:"saga_id"`
	AssetID      uuid.UUID  `db:"asset_id" json:"asset_id"`
	UserID       string     `db:"user_id" json:"user_id"`
	SourcePath   string     `db:"source_path" json:"source_path"`
	OutputPath   string     `db:"output_path" json:"output_path"`
	ContentType  string     `db:"content_type" json:"content_type"`
	Status       JobStatus  `db:"status" json:"status"`
	ErrorMessage string     `db:"error_message" json:"error_message,omitempty"`
	Progress     int        `db:"progress" json:"progress"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	StartedAt    *time.Time `db:"started_at" json:"started_at,omitempty"`
	CompletedAt  *time.Time `db:"completed_at" json:"completed_at,omitempty"`
}
