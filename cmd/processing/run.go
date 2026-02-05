package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/romariotrain/media-platform/internal/processing/ffmpeg"
	"github.com/romariotrain/media-platform/internal/processing/kafka"
	"github.com/romariotrain/media-platform/internal/processing/outbox"
	"github.com/romariotrain/media-platform/internal/processing/service"
	pg "github.com/romariotrain/media-platform/internal/processing/storage/postgres"
	"github.com/rs/zerolog/log"
	kafkago "github.com/segmentio/kafka-go"
)

func run(ctx context.Context) error {
	_ = godotenv.Load()

	// Database
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is empty")
	}

	log.Info().Msg("connecting to database...")
	db, err := pg.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	defer db.Close()
	log.Info().Msg("database connected")

	// Output directory for processed files
	outputDir := os.Getenv("OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "processed"
	}

	// FFmpeg processor (или mock если ffmpeg недоступен)
	var processor service.Processor
	ffmpegProcessor, err := ffmpeg.NewProcessor(outputDir)
	if err != nil {
		log.Warn().Err(err).Msg("ffmpeg not available, using mock processor")
		processor = service.NewMockProcessor(outputDir)
	} else {
		processor = ffmpegProcessor
	}

	// Репозитории
	taskRepo := pg.NewProcessingTaskRepo(db)
	outboxRepo := pg.NewOutboxRepo(db)

	// Сервис
	svc := service.New(taskRepo, outboxRepo, processor)
	log.Info().Msg("processing service initialized")

	// Kafka настройки
	kafkaBrokers := []string{"localhost:9092"}
	kafkaGroupID := "processing-service"

	// Kafka Writer для outbox publisher
	kafkaWriter := &kafkago.Writer{
		Addr:                   kafkago.TCP(kafkaBrokers...),
		Balancer:               &kafkago.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	defer kafkaWriter.Close()

	// Outbox Publisher
	publisher, err := outbox.NewPublisher(outbox.PublisherConfig{
		OutboxRepo: outboxRepo,
		Writer:     kafkaWriter,
		Interval:   5 * time.Second,
		BatchSize:  100,
		Logger:     log.Logger,
	})
	if err != nil {
		return fmt.Errorf("create outbox publisher: %w", err)
	}

	// Kafka Consumer
	consumer := kafka.NewConsumer(kafkaBrokers, kafkaGroupID, svc)
	defer consumer.Close()

	// Запускаем компоненты
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// Outbox Publisher
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("starting outbox publisher...")
		if err := publisher.Start(ctx); err != nil && err != context.Canceled {
			errCh <- fmt.Errorf("outbox publisher: %w", err)
		}
	}()

	// Kafka Consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("starting kafka consumer...")
		if err := consumer.Start(ctx); err != nil && err != context.Canceled {
			errCh <- fmt.Errorf("kafka consumer: %w", err)
		}
	}()

	log.Info().Msg("processing service started successfully")

	// Ждём ошибки или сигнала остановки
	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down gracefully...")
		wg.Wait()
		return nil

	case err := <-errCh:
		return err
	}
}
