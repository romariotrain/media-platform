package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/romariotrain/media-platform/internal/quota/kafka"
	"github.com/romariotrain/media-platform/internal/quota/outbox"
	"github.com/romariotrain/media-platform/internal/quota/service"
	pg "github.com/romariotrain/media-platform/internal/quota/storage/postgres"
	"github.com/rs/zerolog/log"
	kafkago "github.com/segmentio/kafka-go"
)

func run(ctx context.Context) error {
	// Загружаем .env файл
	_ = godotenv.Load()

	// Получаем DATABASE_URL из окружения
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL is empty")
	}

	log.Info().Msg("connecting to database...")
	db, err := pg.ConnectDB(dsn)
	if err != nil {
		return fmt.Errorf("db connect: %w", err)
	}
	defer db.Close()
	log.Info().Msg("database connected")

	// Инициализация репозиториев
	quotaRepo := pg.NewQuotaRepo(db)
	outboxRepo := pg.NewOutboxRepo(db)

	// Инициализация сервиса
	svc := service.New(quotaRepo, outboxRepo)
	log.Info().Msg("service initialized")

	// Kafka настройки
	kafkaBrokers := []string{"localhost:9092"} // TODO: из env
	kafkaGroupID := "quota-service"

	// Kafka Writer для outbox publisher
	kafkaWriter := &kafkago.Writer{
		Addr:                   kafkago.TCP(kafkaBrokers...),
		Balancer:               &kafkago.LeastBytes{},
		AllowAutoTopicCreation: true,
	}
	defer kafkaWriter.Close()

	// Outbox Publisher
	publisher := outbox.NewPublisher(outboxRepo, kafkaWriter)

	// Kafka Consumer
	consumer := kafka.NewConsumer(kafkaBrokers, kafkaGroupID, svc)
	defer consumer.Close()

	// WaitGroup для управления горутинами
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// Запускаем Outbox Publisher
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("starting outbox publisher...")
		if err := publisher.Start(ctx); err != nil && err != context.Canceled {
			errCh <- fmt.Errorf("outbox publisher: %w", err)
		}
	}()

	// Запускаем Kafka Consumer
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Info().Msg("starting kafka consumer...")
		if err := consumer.Start(ctx); err != nil && err != context.Canceled {
			errCh <- fmt.Errorf("kafka consumer: %w", err)
		}
	}()

	log.Info().Msg("quota service started successfully")

	// Ждём либо ошибки, либо сигнала остановки
	select {
	case <-ctx.Done():
		log.Info().Msg("shutting down gracefully...")
		wg.Wait()
		return nil

	case err := <-errCh:
		return err
	}
}
