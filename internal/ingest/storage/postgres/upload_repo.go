package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/romariotrain/media-platform/internal/ingest/models"
)

type UploadSessionRepo struct {
	db *sqlx.DB
}

func NewUploadSessionRepo(db *sqlx.DB) *UploadSessionRepo {
	return &UploadSessionRepo{db: db}
}

func (r *UploadSessionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.UploadSession, error) {
	const q = `
		SELECT id, saga_id, asset_id, user_id, token, status, file_name, file_size,
		       content_type, storage_path, expires_at, created_at, updated_at
		FROM upload_sessions WHERE id = $1
	`
	var s models.UploadSession
	if err := r.db.GetContext(ctx, &s, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("upload session get by id: %w", err)
	}
	return &s, nil
}

func (r *UploadSessionRepo) GetByToken(ctx context.Context, token string) (*models.UploadSession, error) {
	const q = `
		SELECT id, saga_id, asset_id, user_id, token, status, file_name, file_size,
		       content_type, storage_path, expires_at, created_at, updated_at
		FROM upload_sessions WHERE token = $1
	`
	var s models.UploadSession
	if err := r.db.GetContext(ctx, &s, q, token); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("upload session get by token: %w", err)
	}
	return &s, nil
}

func (r *UploadSessionRepo) FindBySagaID(ctx context.Context, sagaID uuid.UUID) (*models.UploadSession, error) {
	const q = `
		SELECT id, saga_id, asset_id, user_id, token, status, file_name, file_size,
		       content_type, storage_path, expires_at, created_at, updated_at
		FROM upload_sessions WHERE saga_id = $1
	`
	var s models.UploadSession
	if err := r.db.GetContext(ctx, &s, q, sagaID); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("upload session find by saga_id: %w", err)
	}
	return &s, nil
}

func (r *UploadSessionRepo) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *UploadSessionRepo) CreateTx(ctx context.Context, tx *sqlx.Tx, s *models.UploadSession) error {
	const q = `
		INSERT INTO upload_sessions (id, saga_id, asset_id, user_id, token, status, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := tx.ExecContext(ctx, q,
		s.ID, s.SagaID, s.AssetID, s.UserID, s.Token, s.Status, s.ExpiresAt, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("upload session create: %w", err)
	}
	return nil
}

func (r *UploadSessionRepo) UpdateStatusTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, status models.UploadStatus) error {
	const q = `UPDATE upload_sessions SET status = $2, updated_at = NOW() WHERE id = $1`
	_, err := tx.ExecContext(ctx, q, id, status)
	if err != nil {
		return fmt.Errorf("upload session update status: %w", err)
	}
	return nil
}

func (r *UploadSessionRepo) UpdateUploadInfoTx(ctx context.Context, tx *sqlx.Tx, id uuid.UUID, fileName string, fileSize int64, contentType string, storagePath string) error {
	const q = `
		UPDATE upload_sessions
		SET file_name = $2, file_size = $3, content_type = $4, storage_path = $5,
		    status = 'uploaded', updated_at = NOW()
		WHERE id = $1
	`
	_, err := tx.ExecContext(ctx, q, id, fileName, fileSize, contentType, storagePath)
	if err != nil {
		return fmt.Errorf("upload session update upload info: %w", err)
	}
	return nil
}
