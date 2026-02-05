package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/processing/storage/postgres"
	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
)

// eventEnvelope — обёртка для события с saga_id для оркестратора
type eventEnvelope struct {
	MessageID string          `json:"message_id"`
	SagaID    uuid.UUID       `json:"saga_id"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// Publisher реализует Outbox паттерн для Processing сервиса
type Publisher struct {
	outboxRepo *postgres.OutboxRepo
	writer     *kafkago.Writer
	interval   time.Duration
	batchSize  int
	logger     zerolog.Logger
}

// PublisherConfig содержит конфигурацию для Publisher
type PublisherConfig struct {
	OutboxRepo *postgres.OutboxRepo
	Writer     *kafkago.Writer
	Interval   time.Duration
	BatchSize  int
	Logger     zerolog.Logger
}

// NewPublisher создаёт новый Outbox Publisher
func NewPublisher(cfg PublisherConfig) (*Publisher, error) {
	if cfg.OutboxRepo == nil {
		return nil, fmt.Errorf("outbox repository is required")
	}
	if cfg.Writer == nil {
		return nil, fmt.Errorf("kafka writer is required")
	}
	if cfg.Interval <= 0 {
		return nil, fmt.Errorf("interval must be positive")
	}
	if cfg.BatchSize <= 0 {
		return nil, fmt.Errorf("batch size must be positive")
	}

	return &Publisher{
		outboxRepo: cfg.OutboxRepo,
		writer:     cfg.Writer,
		interval:   cfg.Interval,
		batchSize:  cfg.BatchSize,
		logger:     cfg.Logger.With().Str("component", "processing_outbox_publisher").Logger(),
	}, nil
}

// Start запускает polling механизм
func (p *Publisher) Start(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.logger.Info().
		Dur("interval", p.interval).
		Int("batch_size", p.batchSize).
		Msg("processing outbox publisher started")

	for {
		select {
		case <-ctx.Done():
			p.logger.Info().Msg("processing outbox publisher stopped")
			return ctx.Err()
		case <-ticker.C:
			if err := p.publishBatch(ctx); err != nil {
				p.logger.Error().Err(err).Msg("failed to publish batch")
			}
		}
	}
}

func (p *Publisher) publishBatch(ctx context.Context) error {
	records, err := p.outboxRepo.GetPending(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("get pending records: %w", err)
	}

	if len(records) == 0 {
		return nil
	}

	p.logger.Info().Int("count", len(records)).Msg("processing batch")

	var (
		published int
		failed    int
		marked    int
	)

	for _, record := range records {
		eventLogger := p.logger.With().
			Str("event_id", record.EventID).
			Str("event_type", record.EventType).
			Int64("outbox_id", record.ID).
			Logger()

		topic := mapEventTypeToTopic(record.EventType)

		// Оборачиваем payload в envelope с saga_id для оркестратора
		var sagaID uuid.UUID
		if record.SagaID != nil {
			sagaID = *record.SagaID
		}
		envelope := eventEnvelope{
			MessageID: record.EventID,
			SagaID:    sagaID,
			Type:      record.EventType,
			Payload:   record.Payload,
		}
		envelopeJSON, err := json.Marshal(envelope)
		if err != nil {
			eventLogger.Error().Err(err).Msg("failed to marshal envelope")
			failed++
			continue
		}

		msg := kafkago.Message{
			Topic: topic,
			Key:   []byte(record.EventID),
			Value: envelopeJSON,
		}

		if err := p.writer.WriteMessages(ctx, msg); err != nil {
			eventLogger.Error().Err(err).Msg("failed to publish event to kafka")
			failed++
			continue
		}

		published++

		if err := p.outboxRepo.MarkProcessed(ctx, record.ID); err != nil {
			eventLogger.Warn().Err(err).Msg("failed to mark event as processed")
		} else {
			marked++
		}
	}

	p.logger.Info().
		Int("total", len(records)).
		Int("published", published).
		Int("failed", failed).
		Int("marked", marked).
		Msg("batch processing completed")

	return nil
}

func mapEventTypeToTopic(eventType string) string {
	switch eventType {
	case "ProcessingSucceeded":
		return "events.processing.succeeded"
	case "ProcessingFailed":
		return "events.processing.failed"
	default:
		return "events.processing.unknown"
	}
}
