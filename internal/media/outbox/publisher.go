package outbox

import (
	"context"
	"log"
	"time"

	"github.com/romariotrain/media-platform/internal/media/kafka"
	"github.com/romariotrain/media-platform/internal/storage/postgres"
)

type Publisher struct {
	outboxRepo *postgres.OutboxRepo
	producer   *kafka.Producer
	interval   time.Duration
	batchSize  int
}

func NewPublisher(
	outboxRepo *postgres.OutboxRepo,
	producer *kafka.Producer,
	interval time.Duration,
	batchSize int,
) *Publisher {
	return &Publisher{
		outboxRepo: outboxRepo,
		producer:   producer,
		interval:   interval,
		batchSize:  batchSize,
	}
}

func (p *Publisher) Start(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	log.Println("Outbox publisher started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Outbox publisher stopped")
			return ctx.Err()

		case <-ticker.C:
			if err := p.publishBatch(ctx); err != nil {
				log.Printf("Error publishing batch: %v", err)
				// Продолжаем работать, не падаем
			}
		}
	}
}

func (p *Publisher) publishBatch(ctx context.Context) error {
	// 1. Читаем pending события
	records, err := p.outboxRepo.GetPending(ctx, p.batchSize)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		return nil // нечего публиковать
	}

	log.Printf("Publishing %d events", len(records))

	// 2. Публикуем каждое событие
	for _, record := range records {
		// Публикуем в Kafka
		err := p.producer.Publish(ctx, record.EventID, record.Payload)
		if err != nil {
			log.Printf("Failed to publish event %s: %v", record.EventID, err)
			continue // пропускаем, попробуем в следующий раз
		}

		// Помечаем как обработанное
		err = p.outboxRepo.MarkProcessed(ctx, record.ID)
		if err != nil {
			log.Printf("Failed to mark event %s as processed: %v", record.EventID, err)
			// Событие опубликовано, но не помечено — оно опубликуется повторно
			// Это нормально для at-least-once delivery
		}
	}

	return nil
}
