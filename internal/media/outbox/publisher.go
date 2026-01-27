package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/romariotrain/media-platform/internal/media/kafka"
	"github.com/romariotrain/media-platform/internal/storage/postgres"
	"github.com/rs/zerolog"
)

// Publisher реализует Outbox паттерн для надёжной публикации событий в Kafka.
// Гарантирует at-least-once delivery семантику.
type Publisher struct {
	outboxRepo *postgres.OutboxRepo
	producer   *kafka.Producer
	interval   time.Duration
	batchSize  int
	logger     zerolog.Logger
}

// PublisherConfig содержит конфигурацию для создания Publisher
type PublisherConfig struct {
	OutboxRepo *postgres.OutboxRepo
	Producer   *kafka.Producer
	Interval   time.Duration
	BatchSize  int
	Logger     zerolog.Logger
}

// NewPublisher создаёт новый экземпляр Publisher с заданной конфигурацией
func NewPublisher(cfg PublisherConfig) (*Publisher, error) {
	if cfg.OutboxRepo == nil {
		return nil, fmt.Errorf("outbox repository is required")
	}
	if cfg.Producer == nil {
		return nil, fmt.Errorf("kafka producer is required")
	}
	if cfg.Interval <= 0 {
		return nil, fmt.Errorf("interval must be positive, got: %v", cfg.Interval)
	}
	if cfg.BatchSize <= 0 {
		return nil, fmt.Errorf("batch size must be positive, got: %d", cfg.BatchSize)
	}

	return &Publisher{
		outboxRepo: cfg.OutboxRepo,
		producer:   cfg.Producer,
		interval:   cfg.Interval,
		batchSize:  cfg.BatchSize,
		logger:     cfg.Logger.With().Str("component", "outbox_publisher").Logger(),
	}, nil
}

// Start запускает polling механизм для обработки событий из outbox таблицы.
// Блокирует до тех пор, пока не будет отменён контекст.
//
// Процесс работы:
// 1. Каждые interval времени проверяет наличие необработанных событий
// 2. Читает batch событий из БД
// 3. Публикует каждое событие в Kafka
// 4. Помечает успешно опубликованные события как processed
//
// Гарантии:
// - At-least-once delivery: события могут быть доставлены повторно
// - Graceful shutdown при отмене контекста
// - Продолжает работу даже при ошибках публикации отдельных событий
func (p *Publisher) Start(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.logger.Info().
		Dur("interval", p.interval).
		Int("batch_size", p.batchSize).
		Msg("outbox publisher started")

	for {
		select {
		case <-ctx.Done():
			p.logger.Info().
				Err(ctx.Err()).
				Msg("outbox publisher stopped")
			return ctx.Err()

		case <-ticker.C:
			if err := p.publishBatch(ctx); err != nil {
				p.logger.Error().
					Err(err).
					Msg("failed to publish batch")
				// Продолжаем работать, не падаем
			}
		}
	}
}

// publishBatch обрабатывает один batch событий из outbox таблицы
func (p *Publisher) publishBatch(ctx context.Context) error {
	// 1. Читаем pending события
	records, err := p.outboxRepo.GetPending(ctx, p.batchSize)
	if err != nil {
		return fmt.Errorf("get pending records: %w", err)
	}

	if len(records) == 0 {
		p.logger.Debug().Msg("no pending events to publish")
		return nil
	}

	p.logger.Info().
		Int("count", len(records)).
		Msg("processing batch")

	// Метрики для tracking
	var (
		published int
		failed    int
		marked    int
	)

	// 2. Публикуем каждое событие
	for _, record := range records {
		eventLogger := p.logger.With().
			Str("event_id", record.EventID).
			Str("event_type", record.EventType).
			Str("aggregate_id", record.AggregateID).
			Int64("outbox_id", record.ID).
			Logger()

		eventLogger.Debug().Msg("publishing event")

		// Публикуем в Kafka
		if err := p.producer.Publish(ctx, record.EventID, record.Payload); err != nil {
			eventLogger.Error().
				Err(err).
				Msg("failed to publish event to kafka")
			failed++
			continue // пропускаем, попробуем в следующий раз
		}

		published++
		eventLogger.Debug().Msg("event published to kafka")

		// Помечаем как обработанное
		if err := p.outboxRepo.MarkProcessed(ctx, record.ID); err != nil {
			eventLogger.Warn().
				Err(err).
				Msg("failed to mark event as processed")
			// Событие опубликовано, но не помечено — оно опубликуется повторно
			// Это нормально для at-least-once delivery
			// Consumer должен быть идемпотентным
		} else {
			marked++
			eventLogger.Debug().Msg("event marked as processed")
		}
	}

	// Итоговая статистика batch
	p.logger.Info().
		Int("total", len(records)).
		Int("published", published).
		Int("failed", failed).
		Int("marked", marked).
		Msg("batch processing completed")

	return nil
}
