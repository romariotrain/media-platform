package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type OutboxRepo struct {
	db *sqlx.DB
}

type OutboxRecord struct {
	ID          int64           `db:"id"`
	EventID     string          `db:"event_id"`
	EventType   string          `db:"event_type"`
	AggregateID string          `db:"aggregate_id"`
	SagaID      *uuid.UUID      `db:"saga_id"`
	Payload     json.RawMessage `db:"payload"`
	OccurredAt  time.Time       `db:"occurred_at"`
}

func NewOutboxRepo(db *sqlx.DB) *OutboxRepo {
	return &OutboxRepo{db: db}
}

// Add вставляет событие в outbox таблицу внутри транзакции
func (r *OutboxRepo) Add(ctx context.Context, tx *sqlx.Tx, eventType string, aggregateID string, sagaID uuid.UUID, payload interface{}) error {
	const query = `
		INSERT INTO ingest_outbox (event_id, event_type, aggregate_id, saga_id, payload, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal event payload: %w", err)
	}

	_, err = tx.ExecContext(ctx, query,
		uuid.New().String(),
		eventType,
		aggregateID,
		sagaID,
		payloadJSON,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert ingest outbox: %w", err)
	}
	return nil
}

// GetPending возвращает необработанные записи из outbox
func (r *OutboxRepo) GetPending(ctx context.Context, limit int) ([]OutboxRecord, error) {
	const q = `
		SELECT id, event_id, event_type, aggregate_id, saga_id, payload, occurred_at
		FROM ingest_outbox
		WHERE processed_at IS NULL
		ORDER BY id ASC
		LIMIT $1
	`
	var records []OutboxRecord
	if err := r.db.SelectContext(ctx, &records, q, limit); err != nil {
		return nil, fmt.Errorf("get pending: %w", err)
	}
	return records, nil
}

// MarkProcessed помечает запись как обработанную
func (r *OutboxRepo) MarkProcessed(ctx context.Context, id int64) error {
	const q = `UPDATE ingest_outbox SET processed_at = NOW() WHERE id = $1`
	_, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	return nil
}
