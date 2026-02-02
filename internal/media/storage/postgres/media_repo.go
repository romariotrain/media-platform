package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/romariotrain/media-platform/internal/media/models"
)

type MediaRepo struct {
	db *sqlx.DB
}

func NewMediaRepo(db *sqlx.DB) *MediaRepo {
	return &MediaRepo{db: db}
}

func (r *MediaRepo) Create(ctx context.Context, m *models.Media) error {
	const q = `
		INSERT INTO media (id, status, type, source, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, q,
		m.ID, m.Status, m.Type, m.Source, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("media create: %w", err)
	}
	return nil
}

func (r *MediaRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Media, error) {
	const q = `
		SELECT id, status, type, source, created_at, updated_at
		FROM media
		WHERE id = $1
	`

	var m models.Media
	if err := r.db.GetContext(ctx, &m, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("media get by id: %w", err)
	}

	return &m, nil
}

func (r *MediaRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.Status) (*models.Media, error) {
	const q = `
		UPDATE media
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id, status, type, source, created_at, updated_at
	`

	var m models.Media
	if err := r.db.GetContext(ctx, &m, q, id, status); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("media update status: %w", err)
	}

	return &m, nil
}

func (r *MediaRepo) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *MediaRepo) UpdateStatusTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, status models.Status) (*models.Media, error) {
	const q = `
        UPDATE media
        SET status = $2, updated_at = NOW()
        WHERE id = $1
        RETURNING id, status, type, source, created_at, updated_at
    `

	var m models.Media
	// Вместо r.db используем tx!
	if err := tx.GetContext(ctx, &m, q, id, status); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("media update status tx: %w", err)
	}

	return &m, nil
}
