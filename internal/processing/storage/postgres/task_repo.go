package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/romariotrain/media-platform/internal/processing/models"
)

type ProcessingTaskRepo struct {
	db *sqlx.DB
}

func NewProcessingTaskRepo(db *sqlx.DB) *ProcessingTaskRepo {
	return &ProcessingTaskRepo{db: db}
}

// taskRow — внутренняя структура для сканирования из БД
type taskRow struct {
	ID           uuid.UUID      `db:"id"`
	SagaID       uuid.UUID      `db:"saga_id"`
	AssetID      uuid.UUID      `db:"asset_id"`
	Status       string         `db:"status"`
	InputPath    string         `db:"input_path"`
	OutputPaths  sql.NullString `db:"output_paths"`
	Metadata     sql.NullString `db:"metadata"`
	ErrorMessage sql.NullString `db:"error_message"`
	StartedAt    sql.NullTime   `db:"started_at"`
	CompletedAt  sql.NullTime   `db:"completed_at"`
	CreatedAt    sql.NullTime   `db:"created_at"`
	UpdatedAt    sql.NullTime   `db:"updated_at"`
}

func (r *taskRow) toModel() (*models.ProcessingTask, error) {
	task := &models.ProcessingTask{
		ID:        r.ID,
		SagaID:    r.SagaID,
		AssetID:   r.AssetID,
		Status:    models.TaskStatus(r.Status),
		InputPath: r.InputPath,
	}

	if r.OutputPaths.Valid && r.OutputPaths.String != "" {
		var out models.OutputPaths
		if err := json.Unmarshal([]byte(r.OutputPaths.String), &out); err == nil {
			task.OutputPaths = &out
		}
	}

	if r.Metadata.Valid && r.Metadata.String != "" {
		var meta models.Metadata
		if err := json.Unmarshal([]byte(r.Metadata.String), &meta); err == nil {
			task.Metadata = &meta
		}
	}

	if r.ErrorMessage.Valid {
		task.ErrorMessage = &r.ErrorMessage.String
	}

	if r.StartedAt.Valid {
		task.StartedAt = &r.StartedAt.Time
	}

	if r.CompletedAt.Valid {
		task.CompletedAt = &r.CompletedAt.Time
	}

	if r.CreatedAt.Valid {
		task.CreatedAt = r.CreatedAt.Time
	}

	if r.UpdatedAt.Valid {
		task.UpdatedAt = r.UpdatedAt.Time
	}

	return task, nil
}

func (r *ProcessingTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.ProcessingTask, error) {
	const q = `
		SELECT id, saga_id, asset_id, status, input_path, output_paths, metadata,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM processing_tasks WHERE id = $1
	`
	var row taskRow
	if err := r.db.GetContext(ctx, &row, q, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("processing task get by id: %w", err)
	}
	return row.toModel()
}

func (r *ProcessingTaskRepo) FindBySagaID(ctx context.Context, sagaID uuid.UUID) (*models.ProcessingTask, error) {
	const q = `
		SELECT id, saga_id, asset_id, status, input_path, output_paths, metadata,
		       error_message, started_at, completed_at, created_at, updated_at
		FROM processing_tasks WHERE saga_id = $1
	`
	var row taskRow
	if err := r.db.GetContext(ctx, &row, q, sagaID); err != nil {
		if err == sql.ErrNoRows {
			return nil, models.ErrNotFound
		}
		return nil, fmt.Errorf("processing task find by saga_id: %w", err)
	}
	return row.toModel()
}

func (r *ProcessingTaskRepo) BeginTx(ctx context.Context) (*sqlx.Tx, error) {
	return r.db.BeginTxx(ctx, nil)
}

func (r *ProcessingTaskRepo) CreateTx(ctx context.Context, tx *sqlx.Tx, task *models.ProcessingTask) error {
	const q = `
		INSERT INTO processing_tasks (id, saga_id, asset_id, status, input_path, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := tx.ExecContext(ctx, q,
		task.ID, task.SagaID, task.AssetID, task.Status, task.InputPath,
		task.StartedAt, task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("processing task create: %w", err)
	}
	return nil
}

func (r *ProcessingTaskRepo) UpdateTx(ctx context.Context, tx *sqlx.Tx, task *models.ProcessingTask) error {
	var outputPathsJSON, metadataJSON []byte
	var err error

	if task.OutputPaths != nil {
		outputPathsJSON, err = json.Marshal(task.OutputPaths)
		if err != nil {
			return fmt.Errorf("marshal output_paths: %w", err)
		}
	}

	if task.Metadata != nil {
		metadataJSON, err = json.Marshal(task.Metadata)
		if err != nil {
			return fmt.Errorf("marshal metadata: %w", err)
		}
	}

	const q = `
		UPDATE processing_tasks
		SET status = $2, output_paths = $3, metadata = $4, error_message = $5,
		    completed_at = $6, updated_at = NOW()
		WHERE id = $1
	`
	_, err = tx.ExecContext(ctx, q,
		task.ID, task.Status, outputPathsJSON, metadataJSON, task.ErrorMessage, task.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("processing task update: %w", err)
	}
	return nil
}
