package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/ingest/models"
	"github.com/romariotrain/media-platform/internal/ingest/service"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Consumer обрабатывает команды из Kafka для Ingest сервиса
type Consumer struct {
	reader  *kafka.Reader
	service *service.Service
}

func NewConsumer(brokers []string, groupID string, svc *service.Service) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    "commands.ingest.prepare",
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
	log.Info().Msg("ingest kafka consumer started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("ingest kafka consumer stopped")
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
				// Не коммитим сообщение при ошибке — будет обработано повторно
				continue
			}

			// Коммитим только после успешной обработки
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Error().Err(err).Msg("failed to commit message")
			}
		}
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg kafka.Message) error {
	// Парсим envelope
	var cmd models.Command
	if err := json.Unmarshal(msg.Value, &cmd); err != nil {
		return fmt.Errorf("unmarshal command: %w", err)
	}

	log.Info().
		Str("message_id", cmd.MessageID).
		Str("saga_id", cmd.SagaID.String()).
		Str("type", string(cmd.Type)).
		Msg("processing ingest command")

	switch cmd.Type {
	case models.CommandPrepareIngest:
		return c.handlePrepareIngest(ctx, cmd)
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

func (c *Consumer) handlePrepareIngest(ctx context.Context, cmd models.Command) error {
	// Парсим payload
	var payload models.PrepareIngestPayload
	payloadJSON, err := json.Marshal(cmd.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return fmt.Errorf("unmarshal prepare ingest payload: %w", err)
	}

	// Валидация
	if payload.AssetID == uuid.Nil || payload.UserID == "" {
		return fmt.Errorf("invalid payload: %+v", payload)
	}

	// Вызываем Service
	session, err := c.service.PrepareUpload(ctx, cmd.SagaID, payload.AssetID, payload.UserID)
	if err != nil {
		return fmt.Errorf("prepare upload: %w", err)
	}

	log.Info().
		Str("session_id", session.ID.String()).
		Str("saga_id", cmd.SagaID.String()).
		Str("token", session.Token).
		Msg("upload session prepared successfully")

	return nil
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
