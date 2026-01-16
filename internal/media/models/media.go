package models

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	UploadedStatus   Status = "uploaded"
	ProcessingStatus Status = "processing"
	ReadyStatus      Status = "ready"
	FailedStatus     Status = "failed"
)

type MediaType string

const (
	Video MediaType = "video"
	Audio MediaType = "audio"
	File  MediaType = "file"
)

type Media struct {
	ID        uuid.UUID `db:"id"`
	Status    Status    `db:"status"`
	Type      MediaType `db:"type"`
	Source    string    `db:"source"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
