package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/processing/models"
	"github.com/romariotrain/media-platform/internal/processing/service"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Consumer обрабатывает команды из Kafka для Processing сервиса
type Consumer struct {
	reader  *kafka.Reader
	service *service.Service
}

func NewConsumer(brokers []string, groupID string, svc *service.Service) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    "commands.processing.start",
		MinBytes: 1,
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		reader:  reader,
		service: svc,
	}
}

// Start запускает обработку команд
func (c *Consumer) Start(ctx context.Context) error {
	log.Info().Msg("processing kafka consumer started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("processing kafka consumer stopped")
			return c.reader.Close()

		default:
			msg, err := c.reader.FetchMessage(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to fetch message")
				continue
			}

			if err := c.handleMessage(ctx, msg); err != nil {
				log.Error().
					Err(err).
					Str("topic", msg.Topic).
					Int64("offset", msg.Offset).
					Msg("failed to handle message")
				// Не коммитим — будет retry
				continue
			}

			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Error().Err(err).Msg("failed to commit message")
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	var cmd models.Command
	if err := json.Unmarshal(msg.Value, &cmd); err != nil {
		return fmt.Errorf("unmarshal command: %w", err)
	}

	log.Info().
		Str("message_id", cmd.MessageID).
		Str("saga_id", cmd.SagaID.String()).
		Str("type", string(cmd.Type)).
		Msg("processing command")

	switch cmd.Type {
	case models.CommandStartProcessing:
		return c.handleStartProcessing(ctx, cmd)
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

func (c *Consumer) handleStartProcessing(ctx context.Context, cmd models.Command) error {
	// Парсим payload
	var payload models.StartProcessingPayload
	payloadJSON, err := json.Marshal(cmd.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return fmt.Errorf("unmarshal start processing payload: %w", err)
	}

	// Валидация
	if payload.AssetID == uuid.Nil || payload.InputPath == "" {
		return fmt.Errorf("invalid payload: %+v", payload)
	}

	// Вызываем Service
	task, err := c.service.ProcessMedia(ctx, cmd.SagaID, payload.AssetID, payload.InputPath)
	if err != nil {
		// Ошибка обработки — логируем но не возвращаем ошибку,
		// т.к. событие ProcessingFailed уже в outbox
		log.Error().
			Err(err).
			Str("saga_id", cmd.SagaID.String()).
			Msg("processing failed")
		return nil
	}

	log.Info().
		Str("task_id", task.ID.String()).
		Str("saga_id", cmd.SagaID.String()).
		Str("status", string(task.Status)).
		Msg("processing completed")

	return nil
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
