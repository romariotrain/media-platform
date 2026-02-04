package models

import (
	"time"

	"github.com/google/uuid"
)

// PublicationStatus представляет статус публикации
type PublicationStatus string

const (
	StatusPending    PublicationStatus = "pending"
	StatusPublishing PublicationStatus = "publishing"
	StatusPublished  PublicationStatus = "published"
	StatusFailed     PublicationStatus = "failed"
)

// SourcePaths содержит пути к исходным файлам (от processing)
type SourcePaths struct {
	Video720p  string `json:"video_720p,omitempty"`
	Video1080p string `json:"video_1080p,omitempty"`
	Thumbnail  string `json:"thumbnail,omitempty"`
}

// PublishedPaths содержит пути к опубликованным файлам
type PublishedPaths struct {
	Video720p  string `json:"video_720p,omitempty"`
	Video1080p string `json:"video_1080p,omitempty"`
	Thumbnail  string `json:"thumbnail,omitempty"`
}

// PublicURLs содержит публичные URL для доступа к медиа
type PublicURLs struct {
	Video720p  string `json:"video_720p,omitempty"`
	Video1080p string `json:"video_1080p,omitempty"`
	Thumbnail  string `json:"thumbnail,omitempty"`
}

// Publication представляет запись о публикации медиафайла
type Publication struct {
	ID             uuid.UUID         `db:"id"              json:"id"`
	SagaID         uuid.UUID         `db:"saga_id"         json:"saga_id"`
	AssetID        uuid.UUID         `db:"asset_id"        json:"asset_id"`
	Status         PublicationStatus `db:"status"          json:"status"`
	SourcePaths    *SourcePaths      `db:"source_paths"    json:"source_paths"`
	PublishedPaths *PublishedPaths   `db:"published_paths" json:"published_paths,omitempty"`
	PublicURLs     *PublicURLs       `db:"public_urls"     json:"public_urls,omitempty"`
	ErrorMessage   *string           `db:"error_message"   json:"error_message,omitempty"`
	PublishedAt    *time.Time        `db:"published_at"    json:"published_at,omitempty"`
	CreatedAt      time.Time         `db:"created_at"      json:"created_at"`
	UpdatedAt      time.Time         `db:"updated_at"      json:"updated_at"`
}
