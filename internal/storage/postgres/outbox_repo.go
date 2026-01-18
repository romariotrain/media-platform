package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/romariotrain/media-platform/internal/media/models"
)

type OutboxRepo struct {
	db *sqlx.DB
}

type OutboxRecord struct {
	ID          int64           `db:"id"`
	EventID     string          `db:"event_id"`
	EventType   string          `db:"event_type"`
	AggregateID string          `db:"aggregate_id"`
	Payload     json.RawMessage `db:"payload"`
	OccurredAt  time.Time       `db:"occurred_at"`
}

func NewOutboxRepo(db *sqlx.DB) *OutboxRepo {
	return &OutboxRepo{db: db}
}

func (r *OutboxRepo) Add(ctx context.Context, tx *sqlx.Tx, event models.DomainEvent) error {
	const query = `
    INSERT INTO outbox (event_id, event_type, aggregate_id, payload, occurred_at)
    VALUES ($1, $2, $3, $4, $5)
`
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	_, err = tx.ExecContext(ctx, query,
		event.EventID(),
		event.EventType(),
		event.AggregateID(),
		payload,
		event.OccurredAt(),
	)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	return nil

}

func (r *OutboxRepo) GetPending(ctx context.Context, limit int) ([]OutboxRecord, error) {
	const q = `
        SELECT id, event_id, event_type, aggregate_id, payload, occurred_at
        FROM outbox
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

func (r *OutboxRepo) MarkProcessed(ctx context.Context, id int64) error {
	const q = `
        UPDATE outbox
        SET processed_at = NOW()
        WHERE id = $1
    `

	_, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}

	return nil
}
