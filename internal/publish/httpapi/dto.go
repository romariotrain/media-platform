package httpapi

import (
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/publish/models"
)

// PublicationResponse — ответ с информацией о публикации
type PublicationResponse struct {
	ID             uuid.UUID              `json:"id"`
	SagaID         uuid.UUID              `json:"saga_id"`
	AssetID        uuid.UUID              `json:"asset_id"`
	Status         string                 `json:"status"`
	SourcePaths    *models.SourcePaths    `json:"source_paths,omitempty"`
	PublishedPaths *models.PublishedPaths  `json:"published_paths,omitempty"`
	PublicURLs     *models.PublicURLs     `json:"public_urls,omitempty"`
	ErrorMessage   *string                `json:"error_message,omitempty"`
	PublishedAt    *time.Time             `json:"published_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

func toPublicationResponse(pub *models.Publication) PublicationResponse {
	return PublicationResponse{
		ID:             pub.ID,
		SagaID:         pub.SagaID,
		AssetID:        pub.AssetID,
		Status:         string(pub.Status),
		SourcePaths:    pub.SourcePaths,
		PublishedPaths: pub.PublishedPaths,
		PublicURLs:     pub.PublicURLs,
		ErrorMessage:   pub.ErrorMessage,
		PublishedAt:    pub.PublishedAt,
		CreatedAt:      pub.CreatedAt,
	}
}

func toPublicationListResponse(pubs []*models.Publication) []PublicationResponse {
	result := make([]PublicationResponse, 0, len(pubs))
	for _, pub := range pubs {
		result = append(result, toPublicationResponse(pub))
	}
	return result
}
