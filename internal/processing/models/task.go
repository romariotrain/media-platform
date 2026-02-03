package models

import (
	"time"

	"github.com/google/uuid"
)

// TaskStatus представляет статус задачи обработки
type TaskStatus string

const (
	StatusPending    TaskStatus = "pending"
	StatusProcessing TaskStatus = "processing"
	StatusCompleted  TaskStatus = "completed"
	StatusFailed     TaskStatus = "failed"
)

// OutputPaths содержит пути к выходным файлам
type OutputPaths struct {
	Video720p  string `json:"video_720p,omitempty"`
	Video1080p string `json:"video_1080p,omitempty"`
	Thumbnail  string `json:"thumbnail,omitempty"`
}

// Metadata содержит метаданные медиафайла
type Metadata struct {
	Duration   float64 `json:"duration,omitempty"`
	Width      int     `json:"width,omitempty"`
	Height     int     `json:"height,omitempty"`
	VideoCodec string  `json:"video_codec,omitempty"`
	AudioCodec string  `json:"audio_codec,omitempty"`
	Bitrate    int64   `json:"bitrate,omitempty"`
	Format     string  `json:"format,omitempty"`
}

// ProcessingTask представляет задачу обработки медиафайла
type ProcessingTask struct {
	ID           uuid.UUID    `db:"id"            json:"id"`
	SagaID       uuid.UUID    `db:"saga_id"       json:"saga_id"`
	AssetID      uuid.UUID    `db:"asset_id"      json:"asset_id"`
	Status       TaskStatus   `db:"status"        json:"status"`
	InputPath    string       `db:"input_path"    json:"input_path"`
	OutputPaths  *OutputPaths `db:"output_paths"  json:"output_paths,omitempty"`
	Metadata     *Metadata    `db:"metadata"      json:"metadata,omitempty"`
	ErrorMessage *string      `db:"error_message" json:"error_message,omitempty"`
	StartedAt    *time.Time   `db:"started_at"    json:"started_at,omitempty"`
	CompletedAt  *time.Time   `db:"completed_at"  json:"completed_at,omitempty"`
	CreatedAt    time.Time    `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at"    json:"updated_at"`
}
