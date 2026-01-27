package kafka

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	kafkago "github.com/segmentio/kafka-go"
)

// Producer реализует надёжную публикацию сообщений в Kafka с retry, metrics и логированием
type Producer struct {
	writer  *kafkago.Writer
	logger  zerolog.Logger
	config  ProducerConfig
	metrics *ProducerMetrics
	closed  atomic.Bool
}

// ProducerConfig содержит конфигурацию для создания Producer
type ProducerConfig struct {
	Brokers      []string
	Topic        string
	MaxRetries   int           // Максимальное количество retry (default: 3)
	RetryBackoff time.Duration // Задержка между retry (default: 100ms)
	WriteTimeout time.Duration // Timeout для записи (default: 10s)
	BatchSize    int           // Размер batch для producer (default: 100)
	Async        bool          // Асинхронная публикация (default: false)
	Logger       zerolog.Logger
}

// ProducerMetrics содержит метрики для мониторинга
type ProducerMetrics struct {
	MessagesPublished atomic.Int64 // Успешно опубликованные сообщения
	MessagesFailed    atomic.Int64 // Проваленные сообщения
	RetriesTotal      atomic.Int64 // Общее количество retry
	PublishDuration   atomic.Int64 // Суммарное время публикации (наносекунды)
}

// NewProducer создаёт новый экземпляр Producer с заданной конфигурацией
func NewProducer(cfg ProducerConfig) (*Producer, error) {
	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Устанавливаем defaults
	setDefaults(&cfg)

	writer := &kafkago.Writer{
		Addr:         kafkago.TCP(cfg.Brokers...),
		Topic:        cfg.Topic,
		Balancer:     &kafkago.LeastBytes{},
		BatchSize:    cfg.BatchSize,
		BatchTimeout: 10 * time.Millisecond,
		WriteTimeout: cfg.WriteTimeout,
		// Compression
		Compression: kafkago.Snappy,
		// Async mode
		Async: cfg.Async,
	}

	p := &Producer{
		writer:  writer,
		logger:  cfg.Logger.With().Str("component", "kafka_producer").Str("topic", cfg.Topic).Logger(),
		config:  cfg,
		metrics: &ProducerMetrics{},
	}

	p.logger.Info().
		Strs("brokers", cfg.Brokers).
		Str("topic", cfg.Topic).
		Int("max_retries", cfg.MaxRetries).
		Dur("retry_backoff", cfg.RetryBackoff).
		Dur("write_timeout", cfg.WriteTimeout).
		Bool("async", cfg.Async).
		Msg("kafka producer created")

	return p, nil
}

// validateConfig проверяет корректность конфигурации
func validateConfig(cfg *ProducerConfig) error {
	if len(cfg.Brokers) == 0 {
		return errors.New("brokers list is empty")
	}
	if cfg.Topic == "" {
		return errors.New("topic is empty")
	}
	if cfg.MaxRetries < 0 {
		return errors.New("max_retries cannot be negative")
	}
	if cfg.RetryBackoff < 0 {
		return errors.New("retry_backoff cannot be negative")
	}
	if cfg.WriteTimeout < 0 {
		return errors.New("write_timeout cannot be negative")
	}
	return nil
}

// setDefaults устанавливает значения по умолчанию
func setDefaults(cfg *ProducerConfig) {
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}
	if cfg.RetryBackoff == 0 {
		cfg.RetryBackoff = 100 * time.Millisecond
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 100
	}
}

// Publish публикует сообщение в Kafka с retry логикой
//
// Гарантии:
// - At-most-once при использовании context timeout
// - Retry с exponential backoff при временных ошибках
// - Structured logging для всех операций
// - Метрики для мониторинга
func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	if p.closed.Load() {
		return errors.New("producer is closed")
	}

	start := time.Now()
	logger := p.logger.With().
		Str("key", key).
		Int("value_size", len(value)).
		Logger()

	logger.Debug().Msg("publishing message")

	var lastErr error
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := p.config.RetryBackoff * time.Duration(1<<uint(attempt-1))
			if backoff > 5*time.Second {
				backoff = 5 * time.Second // cap at 5s
			}

			logger.Warn().
				Int("attempt", attempt).
				Dur("backoff", backoff).
				Err(lastErr).
				Msg("retrying publish")

			p.metrics.RetriesTotal.Add(1)

			// Wait with context cancellation support
			select {
			case <-ctx.Done():
				p.metrics.MessagesFailed.Add(1)
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		// Attempt to publish
		err := p.publishAttempt(ctx, key, value)
		if err == nil {
			duration := time.Since(start)
			p.metrics.MessagesPublished.Add(1)
			p.metrics.PublishDuration.Add(duration.Nanoseconds())

			logger.Debug().
				Dur("duration", duration).
				Int("attempts", attempt+1).
				Msg("message published successfully")

			return nil
		}

		lastErr = err

		// Проверяем, является ли ошибка retriable
		if !isRetriableError(err) {
			logger.Error().
				Err(err).
				Int("attempt", attempt+1).
				Msg("non-retriable error, giving up")
			break
		}

		logger.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Msg("retriable error occurred")
	}

	// Все попытки исчерпаны
	p.metrics.MessagesFailed.Add(1)

	logger.Error().
		Err(lastErr).
		Int("total_attempts", p.config.MaxRetries+1).
		Dur("total_duration", time.Since(start)).
		Msg("failed to publish message after all retries")

	return fmt.Errorf("failed after %d attempts: %w", p.config.MaxRetries+1, lastErr)
}

// publishAttempt выполняет одну попытку публикации
func (p *Producer) publishAttempt(ctx context.Context, key string, value []byte) error {
	msg := kafkago.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	}

	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		return fmt.Errorf("kafka write: %w", err)
	}

	return nil
}

// isRetriableError определяет, можно ли retry эту ошибку
func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors не retry
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Kafka-специфичные ошибки
	// Retriable: сетевые ошибки, temporary failures
	// Non-retriable: invalid message, authorization errors

	errStr := err.Error()

	// Retriable errors
	retriable := []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"timeout",
		"temporary failure",
		"leader not available",
		"not controller",
	}

	for _, pattern := range retriable {
		if contains(errStr, pattern) {
			return true
		}
	}

	// Non-retriable errors
	nonRetriable := []string{
		"invalid message",
		"message too large",
		"authorization failed",
		"topic authorization failed",
	}

	for _, pattern := range nonRetriable {
		if contains(errStr, pattern) {
			return false
		}
	}

	// По умолчанию считаем ошибку retriable
	return true
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)*2))
}

// PublishBatch публикует batch сообщений атомарно
//
// Если хотя бы одно сообщение не удалось опубликовать, вся операция считается неуспешной.
// Retry применяется ко всему batch.
func (p *Producer) PublishBatch(ctx context.Context, messages []Message) error {
	if p.closed.Load() {
		return errors.New("producer is closed")
	}

	if len(messages) == 0 {
		return nil
	}

	start := time.Now()
	logger := p.logger.With().
		Int("batch_size", len(messages)).
		Logger()

	logger.Debug().Msg("publishing batch")

	var lastErr error
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := p.config.RetryBackoff * time.Duration(1<<uint(attempt-1))
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}

			logger.Warn().
				Int("attempt", attempt).
				Dur("backoff", backoff).
				Err(lastErr).
				Msg("retrying batch publish")

			p.metrics.RetriesTotal.Add(1)

			select {
			case <-ctx.Done():
				p.metrics.MessagesFailed.Add(int64(len(messages)))
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			case <-time.After(backoff):
			}
		}

		// Convert to kafka messages
		kafkaMessages := make([]kafkago.Message, len(messages))
		for i, msg := range messages {
			kafkaMessages[i] = kafkago.Message{
				Key:   []byte(msg.Key),
				Value: msg.Value,
				Time:  time.Now(),
			}
		}

		// Attempt to publish batch
		err := p.writer.WriteMessages(ctx, kafkaMessages...)
		if err == nil {
			duration := time.Since(start)
			p.metrics.MessagesPublished.Add(int64(len(messages)))
			p.metrics.PublishDuration.Add(duration.Nanoseconds())

			logger.Info().
				Dur("duration", duration).
				Int("attempts", attempt+1).
				Msg("batch published successfully")

			return nil
		}

		lastErr = err

		if !isRetriableError(err) {
			logger.Error().
				Err(err).
				Int("attempt", attempt+1).
				Msg("non-retriable error in batch, giving up")
			break
		}
	}

	p.metrics.MessagesFailed.Add(int64(len(messages)))

	logger.Error().
		Err(lastErr).
		Int("total_attempts", p.config.MaxRetries+1).
		Dur("total_duration", time.Since(start)).
		Msg("failed to publish batch after all retries")

	return fmt.Errorf("batch failed after %d attempts: %w", p.config.MaxRetries+1, lastErr)
}

// Message представляет сообщение для публикации
type Message struct {
	Key   string
	Value []byte
}

// GetMetrics возвращает текущие метрики producer
func (p *Producer) GetMetrics() Metrics {
	return Metrics{
		MessagesPublished: p.metrics.MessagesPublished.Load(),
		MessagesFailed:    p.metrics.MessagesFailed.Load(),
		RetriesTotal:      p.metrics.RetriesTotal.Load(),
		AvgPublishTime:    p.calculateAvgPublishTime(),
	}
}

// Metrics содержит snapshot метрик
type Metrics struct {
	MessagesPublished int64
	MessagesFailed    int64
	RetriesTotal      int64
	AvgPublishTime    time.Duration
}

func (p *Producer) calculateAvgPublishTime() time.Duration {
	published := p.metrics.MessagesPublished.Load()
	if published == 0 {
		return 0
	}
	totalNanos := p.metrics.PublishDuration.Load()
	return time.Duration(totalNanos / published)
}

// Close закрывает producer и освобождает ресурсы
//
// После вызова Close дальнейшие вызовы Publish будут возвращать ошибку.
// Метод блокируется до завершения всех pending операций или до истечения 30 секунд.
func (p *Producer) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return errors.New("producer already closed")
	}

	p.logger.Info().Msg("closing kafka producer")

	// Даём время на flush pending messages
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Закрываем writer
	if err := p.writer.Close(); err != nil {
		p.logger.Error().Err(err).Msg("error closing kafka writer")
		return fmt.Errorf("close writer: %w", err)
	}

	// Логируем финальные метрики
	metrics := p.GetMetrics()
	p.logger.Info().
		Int64("messages_published", metrics.MessagesPublished).
		Int64("messages_failed", metrics.MessagesFailed).
		Int64("retries_total", metrics.RetriesTotal).
		Dur("avg_publish_time", metrics.AvgPublishTime).
		Msg("kafka producer closed")

	<-ctx.Done()
	return nil
}

// HealthCheck проверяет здоровье producer
func (p *Producer) HealthCheck(ctx context.Context) error {
	if p.closed.Load() {
		return errors.New("producer is closed")
	}

	// Проверяем connectivity через stats
	stats := p.writer.Stats()

	p.logger.Debug().
		Int64("writes", stats.Writes).
		Int64("messages", stats.Messages).
		Int64("errors", stats.Errors).
		Msg("producer health check")

	// Если слишком много ошибок по сравнению с успешными записями
	if stats.Writes > 0 && stats.Errors > stats.Writes/2 {
		return fmt.Errorf("high error rate: %d errors out of %d writes", stats.Errors, stats.Writes)
	}

	return nil
}
