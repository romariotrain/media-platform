package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/romariotrain/media-platform/internal/orchestrator/models"
	"github.com/romariotrain/media-platform/internal/orchestrator/service"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// eventTopics — все топики событий, которые оркестратор слушает
var eventTopics = []string{
	"events.ingest.ready",
	"events.ingest.uploaded",
	"events.processing.succeeded",
	"events.processing.failed",
	"events.publish.succeeded",
	"events.publish.failed",
}

// topicToEventType маппинг топика в тип события
var topicToEventType = map[string]models.IncomingEventType{
	"events.ingest.ready":           models.EventIngestReady,
	"events.ingest.uploaded":        models.EventIngestUploaded,
	"events.processing.succeeded":   models.EventProcessingSucceeded,
	"events.processing.failed":      models.EventProcessingFailed,
	"events.publish.succeeded":      models.EventPublishSucceeded,
	"events.publish.failed":         models.EventPublishFailed,
}

// Consumer обрабатывает события из Kafka для Orchestrator сервиса
type Consumer struct {
	readers []*kafka.Reader
	service *service.Service
}

func NewConsumer(brokers []string, groupID string, svc *service.Service) *Consumer {
	readers := make([]*kafka.Reader, 0, len(eventTopics))
	for _, topic := range eventTopics {
		reader := kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			GroupID:  groupID,
			Topic:    topic,
			MinBytes: 1,
			MaxBytes: 10e6,
		})
		readers = append(readers, reader)
	}

	return &Consumer{
		readers: readers,
		service: svc,
	}
}

// Start запускает обработку событий из всех топиков
func (c *Consumer) Start(ctx context.Context) error {
	log.Info().
		Int("topics", len(c.readers)).
		Msg("orchestrator kafka consumer started")

	errCh := make(chan error, len(c.readers))

	for _, reader := range c.readers {
		go func(r *kafka.Reader) {
			errCh <- c.consumeTopic(ctx, r)
		}(reader)
	}

	// Ждём первую ошибку или завершение контекста
	select {
	case <-ctx.Done():
		log.Info().Msg("orchestrator kafka consumer stopped")
		return nil
	case err := <-errCh:
		return err
	}
}

func (c *Consumer) consumeTopic(ctx context.Context, reader *kafka.Reader) error {
	topic := reader.Config().Topic
	log.Info().Str("topic", topic).Msg("consuming topic")

	for {
		select {
		case <-ctx.Done():
			return reader.Close()

		default:
			msg, err := reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				log.Error().Err(err).Str("topic", topic).Msg("failed to fetch message")
				continue
			}

			if err := c.handleMessage(ctx, msg); err != nil {
				log.Error().
					Err(err).
					Str("topic", msg.Topic).
					Int64("offset", msg.Offset).
					Msg("failed to handle message")
				continue
			}

			if err := reader.CommitMessages(ctx, msg); err != nil {
				log.Error().Err(err).Str("topic", topic).Msg("failed to commit message")
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	// Определяем тип события по топику
	eventType, ok := topicToEventType[msg.Topic]
	if !ok {
		return fmt.Errorf("unknown topic: %s", msg.Topic)
	}

	// Парсим событие
	var event models.IncomingEvent
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	// Устанавливаем тип из маппинга (более надёжно чем из payload)
	event.Type = eventType

	log.Info().
		Str("message_id", event.MessageID).
		Str("saga_id", event.SagaID.String()).
		Str("type", string(event.Type)).
		Str("topic", msg.Topic).
		Msg("orchestrator received event")

	return c.service.HandleEvent(ctx, event)
}

// Close закрывает все consumers
func (c *Consumer) Close() error {
	var lastErr error
	for _, reader := range c.readers {
		if err := reader.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
