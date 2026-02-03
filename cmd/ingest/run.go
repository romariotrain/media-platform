package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/romariotrain/media-platform/internal/ingest/httpapi"
	"github.com/romariotrain/media-platform/internal/ingest/kafka"
	"github.com/romariotrain/media-platform/internal/ingest/outbox"
	"github.com/romariotrain/media-platform/internal/ingest/service"
	"github.com/romariotrain/media-platform/internal/ingest/storage/fs"
	pg "github.com/romariotrain/media-platform/internal/ingest/storage/postgres"
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

	// File store
	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "uploads"
	}
	fileStore, err := fs.NewFileStore(uploadDir)
	if err != nil {
		return fmt.Errorf("create file store: %w", err)
	}

	// Репозитории
	sessionRepo := pg.NewUploadSessionRepo(db)
	outboxRepo := pg.NewOutboxRepo(db)

	// Сервис
	svc := service.New(sessionRepo, outboxRepo, fileStore)
	log.Info().Msg("ingest service initialized")

	// HTTP
	h := httpapi.New(svc)
	router := httpapi.NewRouter(h)

	srv := &http.Server{
		Addr:              ":8082",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Kafka настройки
	kafkaBrokers := []string{"localhost:9092"}
	kafkaGroupID := "ingest-service"

	// Kafka Writer для outbox publisher (topic per message)
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

	// Запускаем все компоненты
	var wg sync.WaitGroup
	errCh := make(chan error, 3)

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

	// HTTP Server
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Str("addr", srv.Addr).Msg("starting HTTP server...")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http server: %w", err)
		}
	}()

	log.Info().Msg("ingest service started successfully")

	// Ждём ошибки или сигнала остановки
	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down gracefully...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("http server shutdown error")
		}

		wg.Wait()
		return nil

	case err := <-errCh:
		return err
	}
}
