package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/romariotrain/media-platform/internal/publish/models"
)

type PublicationRepo struct {
	db *sqlx.DB
}

func NewPublicationRepo(db *sqlx.DB) *PublicationRepo {
	return &PublicationRepo{db: db}
}

// pubRow — внутренняя структура для сканирования из БД
type pubRow struct {
	ID             uuid.UUID      `db:"id"`
	SagaID         uuid.UUID      `db:"saga_id"`
	AssetID        uuid.UUID      `db:"asset_id"`
	Status         string         `db:"status"`
	SourcePaths    sql.NullString `db:"source_paths"`
	PublishedPaths sql.NullString `db:"published_paths"`
	PublicURLs     sql.NullString `db:"public_urls"`
	ErrorMessage   sql.NullString `db:"error_message"`
	PublishedAt    sql.NullTime   `db:"published_at"`
	CreatedAt      sql.NullTime   `db:"created_at"`
	UpdatedAt      sql.NullTime   `db:"updated_at"`
}

func (r *pubRow) toModel() (*models.Publication, error) {
	pub := &models.Publication{
		ID:      r.ID,
		SagaID:  r.SagaID,
		AssetID: r.AssetID,
		Status:  models.PublicationStatus(r.Status),
	}

	if r.SourcePaths.Valid && r.SourcePaths.String != "" {
		var sp models.SourcePaths
		if err := json.Unmarshal([]byte(r.SourcePaths.String), &sp); err == nil {
			pub.SourcePaths = &sp
		}
	}

	if r.PublishedPaths.Valid && r.PublishedPaths.String != "" {
		var pp models.PublishedPaths
		if err := json.Unmarshal([]byte(r.PublishedPaths.String), &pp); err == nil {
			pub.PublishedPaths = &pp
		}
	}

	if r.PublicURLs.Valid && r.PublicURLs.String != "" {
		var urls models.PublicURLs
		if err := json.Unmarshal([]byte(r.PublicURLs.String), &urls); err == nil {
			pub.PublicURLs = &urls
		}
	}

	if r.ErrorMessage.Valid {
		pub.ErrorMessage = &r.ErrorMessage.String
	}

	if r.PublishedAt.Valid {
		pub.PublishedAt = &r.PublishedAt.Time
	}

	if r.CreatedAt.Valid {
		pub.CreatedAt = r.CreatedAt.Time
	}

	if r.UpdatedAt.Valid {
		pub.UpdatedAt = r.UpdatedAt.Time
	}

	return pub, nil
}

const selectCols = `id, saga_id, asset_id, status, source_paths, published_paths, public_urls,
       error_message, published_at, created_at, updated_at`

func (r *PublicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Publication, error) {
	q := `SELECT ` + selectCols + ` FROM publications WHERE id = $1`
	var row pubRow
	if err := r.db.GetContext(ctx, &row, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("publication get by id: %w", err)
	}
	return row.toModel()
}

func (r *PublicationRepo) FindBySagaID(ctx context.Context, sagaID uuid.UUID) (*models.Publication, error) {
	q := `SELECT ` + selectCols + ` FROM publications WHERE saga_id = $1`
	var row pubRow
	if err := r.db.GetContext(ctx, &row, q, sagaID); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("publication find by saga_id: %w", err)
	}
	return row.toModel()
}

func (r *PublicationRepo) ListByAssetID(ctx context.Context, assetID uuid.UUID) ([]*models.Publication, error) {
	q := `SELECT ` + selectCols + ` FROM publications WHERE asset_id = $1 ORDER BY created_at DESC`
	var rows []pubRow
	if err := r.db.SelectContext(ctx, &rows, q, assetID); err != nil {
		return nil, fmt.Errorf("publication list by asset_id: %w", err)
	}

	pubs := make([]*models.Publication, 0, len(rows))
	for _, row := range rows {
		pub, err := row.toModel()
		if err != nil {
			return nil, err
		}
		pubs = append(pubs, pub)
	}
	return pubs, nil
}

func (r *PublicationRepo) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *PublicationRepo) CreateTx(ctx context.Context, tx *sqlx.Tx, pub *models.Publication) error {
	sourcePathsJSON, err := json.Marshal(pub.SourcePaths)
	if err != nil {
		return fmt.Errorf("marshal source_paths: %w", err)
	}

	const q = `
		INSERT INTO publications (id, saga_id, asset_id, status, source_paths, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = tx.ExecContext(ctx, q,
		pub.ID, pub.SagaID, pub.AssetID, pub.Status, sourcePathsJSON,
		pub.CreatedAt, pub.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("publication create: %w", err)
	}
	return nil
}

func (r *PublicationRepo) UpdateTx(ctx context.Context, tx *sqlx.Tx, pub *models.Publication) error {
	var publishedPathsJSON, publicURLsJSON []byte
	var err error

	if pub.PublishedPaths != nil {
		publishedPathsJSON, err = json.Marshal(pub.PublishedPaths)
		if err != nil {
			return fmt.Errorf("marshal published_paths: %w", err)
		}
	}

	if pub.PublicURLs != nil {
		publicURLsJSON, err = json.Marshal(pub.PublicURLs)
		if err != nil {
			return fmt.Errorf("marshal public_urls: %w", err)
		}
	}

	const q = `
		UPDATE publications
		SET status = $2, published_paths = $3, public_urls = $4, error_message = $5,
		    published_at = $6, updated_at = NOW()
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, q,
		pub.ID, pub.Status, publishedPathsJSON, publicURLsJSON, pub.ErrorMessage, pub.PublishedAt,
	)
	if err != nil {
		return fmt.Errorf("publication update: %w", err)
	}
	return nil
}
