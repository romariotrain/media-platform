package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/romariotrain/media-platform/internal/quota/models"
	"github.com/romariotrain/media-platform/internal/quota/service"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
)

// Consumer обрабатывает команды из Kafka
type Consumer struct {
	reader  *kafka.Reader
	service *service.Service
	// TODO: добавить idempotency checker (Redis)
}

func NewConsumer(brokers []string, groupID string, svc *service.Service) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  groupID,
		Topic:    "commands.quota.reserve", // слушаем все команды для quota
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
	log.Info().Msg("kafka consumer started")

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("kafka consumer stopped")
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
				// Не коммитим сообщение при ошибке - оно будет обработано повторно
				continue
			}

			// Коммитим сообщение только после успешной обработки
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
		Msg("processing command")

	// TODO: Проверка идемпотентности через Redis
	// if alreadyProcessed(cmd.MessageID) {
	//     log.Info().Msg("command already processed, skipping")
	//     return nil
	// }

	// Обрабатываем команду в зависимости от типа
	switch cmd.Type {
	case models.CommandReserveQuota:
		return c.handleReserveQuota(ctx, cmd)
	case models.CommandReleaseQuota:
		return c.handleReleaseQuota(ctx, cmd)
	default:
		return fmt.Errorf("unknown command type: %s", cmd.Type)
	}
}

func (c *Consumer) handleReserveQuota(ctx context.Context, cmd models.Command) error {
	// Парсим payload
	var payload models.ReserveQuotaPayload
	payloadJSON, err := json.Marshal(cmd.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return fmt.Errorf("unmarshal reserve quota payload: %w", err)
	}

	// Валидация
	if payload.UserID == "" || payload.AssetID == uuid.Nil || payload.Amount <= 0 {
		return fmt.Errorf("invalid payload: %+v", payload)
	}

	// Преобразуем строку типа в QuotaType
	var quotaType models.QuotaType
	switch payload.Type {
	case "storage":
		quotaType = models.StorageQuota
	case "count":
		quotaType = models.CountQuota
	default:
		return fmt.Errorf("unknown quota type: %s", payload.Type)
	}

	// Вызываем Service
	quota, err := c.service.ReserveQuota(
		ctx,
		cmd.SagaID,
		payload.UserID,
		payload.AssetID,
		quotaType,
		payload.Amount,
	)

	if err != nil {
		// Проверяем тип ошибки
		if err == models.ErrInsufficientQuota {
			log.Warn().
				Str("saga_id", cmd.SagaID.String()).
				Str("user_id", payload.UserID).
				Int64("requested", payload.Amount).
				Msg("insufficient quota")

			// TODO: публиковать событие QuotaFailed напрямую в Kafka
			// (не через outbox, т.к. транзакция откатилась)
			return nil // не возвращаем ошибку, чтобы закоммитить сообщение
		}

		return fmt.Errorf("reserve quota: %w", err)
	}

	log.Info().
		Str("quota_id", quota.ID.String()).
		Str("saga_id", cmd.SagaID.String()).
		Int64("amount", quota.Amount).
		Msg("quota reserved successfully")

	// Событие QuotaReserved будет опубликовано через outbox publisher
	return nil
}

func (c *Consumer) handleReleaseQuota(ctx context.Context, cmd models.Command) error {
	// Парсим payload
	var payload models.ReleaseQuotaPayload
	payloadJSON, err := json.Marshal(cmd.Payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return fmt.Errorf("unmarshal release quota payload: %w", err)
	}

	// Валидация
	if payload.AssetID == uuid.Nil {
		return fmt.Errorf("invalid payload: %+v", payload)
	}

	// Вызываем Service
	quota, err := c.service.ReleaseQuota(ctx, payload.AssetID)
	if err != nil {
		// Если квота не найдена - это не критично для компенсации
		// (возможно резервация вообще не прошла)
		if err == models.ErrNotFound {
			log.Warn().
				Str("saga_id", cmd.SagaID.String()).
				Str("asset_id", payload.AssetID.String()).
				Msg("quota not found for release (maybe never reserved)")
			return nil
		}

		return fmt.Errorf("release quota: %w", err)
	}

	log.Info().
		Str("quota_id", quota.ID.String()).
		Str("saga_id", cmd.SagaID.String()).
		Msg("quota released successfully")

	// Событие QuotaReleased будет опубликовано через outbox publisher
	return nil
}

// Close закрывает consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
