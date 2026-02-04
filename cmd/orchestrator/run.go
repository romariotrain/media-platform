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
	"github.com/romariotrain/media-platform/internal/orchestrator/httpapi"
	"github.com/romariotrain/media-platform/internal/orchestrator/kafka"
	"github.com/romariotrain/media-platform/internal/orchestrator/outbox"
	"github.com/romariotrain/media-platform/internal/orchestrator/service"
	pg "github.com/romariotrain/media-platform/internal/orchestrator/storage/postgres"
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

	// Репозитории
	sagaRepo := pg.NewSagaRepo(db)
	outboxRepo := pg.NewOutboxRepo(db)

	// Сервис
	svc := service.New(sagaRepo, outboxRepo)
	log.Info().Msg("orchestrator service initialized")

	// HTTP
	h := httpapi.New(svc)
	router := httpapi.NewRouter(h)

	srv := &http.Server{
		Addr:              ":8084",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Kafka настройки
	kafkaBrokers := []string{"localhost:9092"}
	kafkaGroupID := "orchestrator-service"

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
		Interval:   3 * time.Second,
		BatchSize:  100,
		Logger:     log.Logger,
	})
	if err != nil {
		return fmt.Errorf("create outbox publisher: %w", err)
	}

	// Kafka Consumer (слушает все event-топики)
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

	log.Info().Msg("orchestrator service started successfully")

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
