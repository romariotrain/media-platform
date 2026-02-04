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
	"github.com/romariotrain/media-platform/internal/publish/httpapi"
	"github.com/romariotrain/media-platform/internal/publish/kafka"
	"github.com/romariotrain/media-platform/internal/publish/outbox"
	"github.com/romariotrain/media-platform/internal/publish/service"
	"github.com/romariotrain/media-platform/internal/publish/storage/fs"
	pg "github.com/romariotrain/media-platform/internal/publish/storage/postgres"
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

	// File publisher
	publishDir := os.Getenv("PUBLISH_DIR")
	if publishDir == "" {
		publishDir = "published"
	}
	baseURL := os.Getenv("PUBLISH_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8083/static"
	}
	filePublisher, err := fs.NewFilePublisher(publishDir, baseURL)
	if err != nil {
		return fmt.Errorf("create file publisher: %w", err)
	}

	// Репозитории
	pubRepo := pg.NewPublicationRepo(db)
	outboxRepo := pg.NewOutboxRepo(db)

	// Сервис
	svc := service.New(pubRepo, outboxRepo, filePublisher)
	log.Info().Msg("publish service initialized")

	// HTTP
	h := httpapi.New(svc)
	router := httpapi.NewRouter(h)

	srv := &http.Server{
		Addr:              ":8083",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Kafka настройки
	kafkaBrokers := []string{"localhost:9092"}
	kafkaGroupID := "publish-service"

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

	log.Info().Msg("publish service started successfully")

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
