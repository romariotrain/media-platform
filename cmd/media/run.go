package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	httpapi "github.com/romariotrain/media-platform/internal/media/httpapi"
	"github.com/romariotrain/media-platform/internal/media/kafka"
	"github.com/romariotrain/media-platform/internal/media/outbox"
	"github.com/romariotrain/media-platform/internal/media/service"

	pg "github.com/romariotrain/media-platform/internal/media/storage/postgres"
	repos "github.com/romariotrain/media-platform/internal/media/storage/postgres"
)

func run(ctx context.Context) error {
	_ = godotenv.Load()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is empty")
	}

	db, err := pg.Connect(ctx, dsn)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	defer db.Close()

	// Dependencies
	mediaRepo := repos.NewMediaRepo(db)
	outboxRepo := repos.NewOutboxRepo(db)

	svc := service.New(mediaRepo, outboxRepo)
	h := httpapi.New(svc)
	router := httpapi.NewRouter(h)

	srv := &http.Server{
		Addr:              ":8081",
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	kafkaProducer, err := kafka.NewProducer(kafka.ProducerConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "events.media",
	})
	if err != nil {
		return fmt.Errorf("create kafka producer: %w", err)
	}
	defer kafkaProducer.Close()

	// Создаём outbox publisher
	outboxPublisher, err := outbox.NewPublisher(outbox.PublisherConfig{
		OutboxRepo: outboxRepo,
		Producer:   kafkaProducer,
		Interval:   5 * time.Second,
		BatchSize:  100,
	})
	if err != nil {
		return fmt.Errorf("create outbox publisher: %w", err)
	}

	// Запускаем publisher в отдельной горутине
	go func() {
		if err := outboxPublisher.Start(ctx); err != nil {
			log.Printf("Outbox publisher error: %v", err)
		}
	}()

	errCh := make(chan error, 1)

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}
		return nil

	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen and serve: %w", err)
	}
}
