package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/romariotrain/media-platform/internal/orchestrator/models"
)

type SagaRepo struct {
	db *sqlx.DB
}

func NewSagaRepo(db *sqlx.DB) *SagaRepo {
	return &SagaRepo{db: db}
}

// sagaRow — внутренняя структура для сканирования из БД
type sagaRow struct {
	ID           uuid.UUID      `db:"id"`
	AssetID      uuid.UUID      `db:"asset_id"`
	UserID       string         `db:"user_id"`
	Status       string         `db:"status"`
	CurrentStep  string         `db:"current_step"`
	Payload      sql.NullString `db:"payload"`
	ErrorMessage sql.NullString `db:"error_message"`
	StartedAt    sql.NullTime   `db:"started_at"`
	CompletedAt  sql.NullTime   `db:"completed_at"`
	CreatedAt    sql.NullTime   `db:"created_at"`
	UpdatedAt    sql.NullTime   `db:"updated_at"`
}

func (r *sagaRow) toModel() (*models.Saga, error) {
	saga := &models.Saga{
		ID:          r.ID,
		AssetID:     r.AssetID,
		UserID:      r.UserID,
		Status:      models.SagaStatus(r.Status),
		CurrentStep: models.SagaStep(r.CurrentStep),
	}

	if r.Payload.Valid && r.Payload.String != "" {
		var p models.SagaPayload
		if err := json.Unmarshal([]byte(r.Payload.String), &p); err == nil {
			saga.Payload = &p
		}
	}

	if r.ErrorMessage.Valid {
		saga.ErrorMessage = &r.ErrorMessage.String
	}

	if r.StartedAt.Valid {
		saga.StartedAt = &r.StartedAt.Time
	}

	if r.CompletedAt.Valid {
		saga.CompletedAt = &r.CompletedAt.Time
	}

	if r.CreatedAt.Valid {
		saga.CreatedAt = r.CreatedAt.Time
	}

	if r.UpdatedAt.Valid {
		saga.UpdatedAt = r.UpdatedAt.Time
	}

	return saga, nil
}

const selectCols = `id, asset_id, user_id, status, current_step, payload,
       error_message, started_at, completed_at, created_at, updated_at`

func (r *SagaRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Saga, error) {
	q := `SELECT ` + selectCols + ` FROM sagas WHERE id = $1`
	var row sagaRow
	if err := r.db.GetContext(ctx, &row, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("saga get by id: %w", err)
	}
	return row.toModel()
}

func (r *SagaRepo) ListByUserID(ctx context.Context, userID string) ([]*models.Saga, error) {
	q := `SELECT ` + selectCols + ` FROM sagas WHERE user_id = $1 ORDER BY created_at DESC`
	var rows []sagaRow
	if err := r.db.SelectContext(ctx, &rows, q, userID); err != nil {
		return nil, fmt.Errorf("saga list by user_id: %w", err)
	}

	sagas := make([]*models.Saga, 0, len(rows))
	for _, row := range rows {
		s, err := row.toModel()
		if err != nil {
			return nil, err
		}
		sagas = append(sagas, s)
	}
	return sagas, nil
}

func (r *SagaRepo) ListByAssetID(ctx context.Context, assetID uuid.UUID) ([]*models.Saga, error) {
	q := `SELECT ` + selectCols + ` FROM sagas WHERE asset_id = $1 ORDER BY created_at DESC`
	var rows []sagaRow
	if err := r.db.SelectContext(ctx, &rows, q, assetID); err != nil {
		return nil, fmt.Errorf("saga list by asset_id: %w", err)
	}

	sagas := make([]*models.Saga, 0, len(rows))
	for _, row := range rows {
		s, err := row.toModel()
		if err != nil {
			return nil, err
		}
		sagas = append(sagas, s)
	}
	return sagas, nil
}

func (r *SagaRepo) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *SagaRepo) CreateTx(ctx context.Context, tx *sqlx.Tx, saga *models.Saga) error {
	var payloadJSON []byte
	var err error

	if saga.Payload != nil {
		payloadJSON, err = json.Marshal(saga.Payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
	}

	const q = `
		INSERT INTO sagas (id, asset_id, user_id, status, current_step, payload, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.ExecContext(ctx, q,
		saga.ID, saga.AssetID, saga.UserID, saga.Status, saga.CurrentStep,
		payloadJSON, saga.StartedAt, saga.CreatedAt, saga.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("saga create: %w", err)
	}
	return nil
}

func (r *SagaRepo) UpdateTx(ctx context.Context, tx *sqlx.Tx, saga *models.Saga) error {
	var payloadJSON []byte
	var err error

	if saga.Payload != nil {
		payloadJSON, err = json.Marshal(saga.Payload)
		if err != nil {
			return fmt.Errorf("marshal payload: %w", err)
		}
	}

	const q = `
		UPDATE sagas
		SET status = $2, current_step = $3, payload = $4, error_message = $5,
		    started_at = $6, completed_at = $7, updated_at = NOW()
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, q,
		saga.ID, saga.Status, saga.CurrentStep, payloadJSON,
		saga.ErrorMessage, saga.StartedAt, saga.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("saga update: %w", err)
	}
	return nil
}
