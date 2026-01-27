package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProducer_Success(t *testing.T) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test-topic",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)

	require.NoError(t, err)
	assert.NotNil(t, producer)
	assert.Equal(t, "test-topic", producer.config.Topic)
	assert.Equal(t, 3, producer.config.MaxRetries) // default
	assert.Equal(t, 100*time.Millisecond, producer.config.RetryBackoff)
}

func TestNewProducer_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  ProducerConfig
		wantErr string
	}{
		{
			name: "empty brokers",
			config: ProducerConfig{
				Brokers: []string{},
				Topic:   "test",
				Logger:  zerolog.Nop(),
			},
			wantErr: "brokers list is empty",
		},
		{
			name: "empty topic",
			config: ProducerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "",
				Logger:  zerolog.Nop(),
			},
			wantErr: "topic is empty",
		},
		{
			name: "negative max retries",
			config: ProducerConfig{
				Brokers:    []string{"localhost:9092"},
				Topic:      "test",
				MaxRetries: -1,
				Logger:     zerolog.Nop(),
			},
			wantErr: "max_retries cannot be negative",
		},
		{
			name: "negative retry backoff",
			config: ProducerConfig{
				Brokers:      []string{"localhost:9092"},
				Topic:        "test",
				RetryBackoff: -1 * time.Second,
				Logger:       zerolog.Nop(),
			},
			wantErr: "retry_backoff cannot be negative",
		},
		{
			name: "negative write timeout",
			config: ProducerConfig{
				Brokers:      []string{"localhost:9092"},
				Topic:        "test",
				WriteTimeout: -1 * time.Second,
				Logger:       zerolog.Nop(),
			},
			wantErr: "write_timeout cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer, err := NewProducer(tt.config)

			require.Error(t, err)
			assert.Nil(t, producer)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestProducer_Defaults(t *testing.T) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)

	assert.Equal(t, 3, producer.config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, producer.config.RetryBackoff)
	assert.Equal(t, 10*time.Second, producer.config.WriteTimeout)
	assert.Equal(t, 100, producer.config.BatchSize)
	assert.False(t, producer.config.Async)
}

func TestProducer_CustomConfig(t *testing.T) {
	cfg := ProducerConfig{
		Brokers:      []string{"localhost:9092"},
		Topic:        "test",
		MaxRetries:   5,
		RetryBackoff: 200 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
		BatchSize:    50,
		Async:        true,
		Logger:       zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)

	assert.Equal(t, 5, producer.config.MaxRetries)
	assert.Equal(t, 200*time.Millisecond, producer.config.RetryBackoff)
	assert.Equal(t, 5*time.Second, producer.config.WriteTimeout)
	assert.Equal(t, 50, producer.config.BatchSize)
	assert.True(t, producer.config.Async)
}

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retriable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retriable: false,
		},
		{
			name:      "context canceled",
			err:       context.Canceled,
			retriable: false,
		},
		{
			name:      "context deadline exceeded",
			err:       context.DeadlineExceeded,
			retriable: false,
		},
		{
			name:      "connection refused",
			err:       errors.New("connection refused"),
			retriable: true,
		},
		{
			name:      "connection reset",
			err:       errors.New("connection reset by peer"),
			retriable: true,
		},
		{
			name:      "timeout",
			err:       errors.New("i/o timeout"),
			retriable: true,
		},
		{
			name:      "leader not available",
			err:       errors.New("leader not available"),
			retriable: true,
		},
		{
			name:      "invalid message",
			err:       errors.New("invalid message format"),
			retriable: false,
		},
		{
			name:      "message too large",
			err:       errors.New("message too large"),
			retriable: false,
		},
		{
			name:      "authorization failed",
			err:       errors.New("authorization failed"),
			retriable: false,
		},
		{
			name:      "unknown error (default retriable)",
			err:       errors.New("some random error"),
			retriable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetriableError(tt.err)
			assert.Equal(t, tt.retriable, result)
		})
	}
}

func TestProducer_GetMetrics(t *testing.T) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)

	// Начальные метрики
	metrics := producer.GetMetrics()
	assert.Equal(t, int64(0), metrics.MessagesPublished)
	assert.Equal(t, int64(0), metrics.MessagesFailed)
	assert.Equal(t, int64(0), metrics.RetriesTotal)

	// Имитируем публикацию
	producer.metrics.MessagesPublished.Add(10)
	producer.metrics.MessagesFailed.Add(2)
	producer.metrics.RetriesTotal.Add(5)
	producer.metrics.PublishDuration.Add(int64(100 * time.Millisecond))

	metrics = producer.GetMetrics()
	assert.Equal(t, int64(10), metrics.MessagesPublished)
	assert.Equal(t, int64(2), metrics.MessagesFailed)
	assert.Equal(t, int64(5), metrics.RetriesTotal)
	assert.Equal(t, 10*time.Millisecond, metrics.AvgPublishTime)
}

func TestProducer_GetMetrics_NoPublished(t *testing.T) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)

	producer.metrics.PublishDuration.Add(int64(100 * time.Millisecond))

	metrics := producer.GetMetrics()
	assert.Equal(t, time.Duration(0), metrics.AvgPublishTime)
}

func TestProducer_Close(t *testing.T) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)

	// First close should succeed (note: will error in test env without real Kafka)
	err = producer.Close()
	// В тестовом окружении может быть ошибка, но главное что closed = true
	assert.True(t, producer.closed.Load())

	// Second close should fail
	err = producer.Close()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already closed")
}

func TestProducer_PublishAfterClose(t *testing.T) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)

	producer.closed.Store(true)

	err = producer.Publish(context.Background(), "test-key", []byte("test-value"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "producer is closed")
}

func TestProducer_PublishBatchAfterClose(t *testing.T) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)

	producer.closed.Store(true)

	messages := []Message{
		{Key: "key1", Value: []byte("value1")},
		{Key: "key2", Value: []byte("value2")},
	}

	err = producer.PublishBatch(context.Background(), messages)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "producer is closed")
}

func TestProducer_PublishBatch_EmptyMessages(t *testing.T) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)

	err = producer.PublishBatch(context.Background(), []Message{})
	assert.NoError(t, err)
}

func TestProducer_HealthCheck_Closed(t *testing.T) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(t, err)

	producer.closed.Store(true)

	err = producer.HealthCheck(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "producer is closed")
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ProducerConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: ProducerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "test",
			},
			wantErr: false,
		},
		{
			name: "empty brokers",
			config: ProducerConfig{
				Brokers: []string{},
				Topic:   "test",
			},
			wantErr: true,
		},
		{
			name: "empty topic",
			config: ProducerConfig{
				Brokers: []string{"localhost:9092"},
				Topic:   "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(&tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSetDefaults(t *testing.T) {
	cfg := ProducerConfig{}
	setDefaults(&cfg)

	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.RetryBackoff)
	assert.Equal(t, 10*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 100, cfg.BatchSize)
}

func TestSetDefaults_DoesNotOverrideExisting(t *testing.T) {
	cfg := ProducerConfig{
		MaxRetries:   5,
		RetryBackoff: 200 * time.Millisecond,
		WriteTimeout: 5 * time.Second,
		BatchSize:    50,
	}
	setDefaults(&cfg)

	assert.Equal(t, 5, cfg.MaxRetries)
	assert.Equal(t, 200*time.Millisecond, cfg.RetryBackoff)
	assert.Equal(t, 5*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 50, cfg.BatchSize)
}

// Benchmark для измерения производительности
func BenchmarkProducer_GetMetrics(b *testing.B) {
	cfg := ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "test",
		Logger:  zerolog.Nop(),
	}

	producer, err := NewProducer(cfg)
	require.NoError(b, err)

	producer.metrics.MessagesPublished.Add(1000)
	producer.metrics.PublishDuration.Add(int64(1000 * time.Millisecond))

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = producer.GetMetrics()
	}
}
