package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/romariotrain/media-platform/internal/orchestrator/storage/postgres"
	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
)

// Publisher реализует Outbox паттерн для Orchestrator сервиса
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
		logger:     cfg.Logger.With().Str("component", "orchestrator_outbox_publisher").Logger(),
	}, nil
}

// Start запускает polling механизм
func (p *Publisher) Start(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.logger.Info().
		Dur("interval", p.interval).
		Int("batch_size", p.batchSize).
		Msg("orchestrator outbox publisher started")

	for {
		select {
		case <-ctx.Done():
			p.logger.Info().Msg("orchestrator outbox publisher stopped")
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

	p.logger.Info().Int("count", len(records)).Msg("publishing batch")

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

		topic := mapCommandTypeToTopic(record.EventType)

		msg := kafkago.Message{
			Topic: topic,
			Key:   []byte(record.EventID),
			Value: record.Payload,
		}

		if err := p.writer.WriteMessages(ctx, msg); err != nil {
			eventLogger.Error().Err(err).Msg("failed to publish command to kafka")
			failed++
			continue
		}

		published++

		if err := p.outboxRepo.MarkProcessed(ctx, record.ID); err != nil {
			eventLogger.Warn().Err(err).Msg("failed to mark command as processed")
		} else {
			marked++
		}
	}

	p.logger.Info().
		Int("total", len(records)).
		Int("published", published).
		Int("failed", failed).
		Int("marked", marked).
		Msg("batch publishing completed")

	return nil
}

func mapCommandTypeToTopic(commandType string) string {
	switch commandType {
	case "PrepareIngest":
		return "commands.ingest.prepare"
	case "StartProcessing":
		return "commands.processing.start"
	case "StartPublish":
		return "commands.publish.start"
	default:
		return "commands.orchestrator.unknown"
	}
}
