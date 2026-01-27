# Outbox Pattern — Архитектура и Реализация

## 📋 Содержание

1. [Что такое Outbox Pattern](#что-такое-outbox-pattern)
2. [Зачем он нужен](#зачем-он-нужен)
3. [Как это работает](#как-это-работает)
4. [Архитектура в нашем проекте](#архитектура-в-нашем-проекте)
5. [Гарантии и ограничения](#гарантии-и-ограничения)
6. [Примеры использования](#примеры-использования)
7. [Мониторинг и troubleshooting](#мониторинг-и-troubleshooting)

---

## Что такое Outbox Pattern

**Outbox Pattern** — это паттерн проектирования для надёжной публикации событий в распределённых системах.

### Проблема

Представьте типичную ситуацию в микросервисной архитектуре:

```go
// ❌ Проблемный код - может потерять событие
func (s *Service) CreateMedia(ctx context.Context, media Media) error {
    // 1. Сохраняем в БД
    if err := s.repo.Save(ctx, media); err != nil {
        return err
    }
    
    // 2. Публикуем событие в Kafka
    event := MediaCreatedEvent{ID: media.ID}
    if err := s.kafka.Publish(ctx, event); err != nil {
        // ⚠️ Что делать? 
        // - Данные сохранены в БД
        // - Событие НЕ опубликовано
        // - Система в несогласованном состоянии!
        return err
    }
    
    return nil
}
```

**Проблемы этого подхода:**

1. **Потеря событий**: если Kafka недоступен или падает сеть после сохранения в БД
2. **Дублирование событий**: если публикация в Kafka прошла, но мы получили ошибку сети
3. **Нет атомарности**: нельзя откатить сохранение в БД, если Kafka недоступен

### Решение

Outbox Pattern решает эту проблему через **атомарное сохранение данных и событий в одной транзакции БД**.

---

## Зачем он нужен

### Основные преимущества

1. ✅ **Атомарность** — данные и события сохраняются в одной транзакции
2. ✅ **Надёжность** — события никогда не теряются
3. ✅ **Отказоустойчивость** — работает даже при временной недоступности Kafka
4. ✅ **Гарантированная доставка** — событие точно будет опубликовано (at-least-once)

### Когда использовать

- ✅ Критичные события, которые не должны быть потеряны
- ✅ Event-driven архитектура с Kafka/RabbitMQ
- ✅ Saga паттерн с координацией между сервисами
- ✅ Audit log и трассировка событий

### Когда НЕ использовать

- ❌ Некритичные события (метрики, аналитика)
- ❌ Требуется exactly-once delivery (Outbox даёт at-least-once)
- ❌ Очень высокая нагрузка (> 10K events/sec) — могут быть проблемы с БД

---

## Как это работает

### Схема работы

```
┌─────────────────────────────────────────────────────────────────┐
│                        MEDIA SERVICE                            │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. HTTP Request                                                │
│     POST /media                                                 │
│        ↓                                                        │
│                                                                 │
│  2. BEGIN TRANSACTION                                           │
│     ┌─────────────────────────────────────────┐                │
│     │  INSERT INTO media (...)                │                │
│     │  VALUES (...);                          │                │
│     │                                          │                │
│     │  INSERT INTO outbox (                   │                │
│     │    event_id,                            │                │
│     │    event_type,                          │                │
│     │    aggregate_id,                        │                │
│     │    payload,                             │                │
│     │    occurred_at                          │                │
│     │  ) VALUES (...);                        │                │
│     └─────────────────────────────────────────┘                │
│     COMMIT                                                      │
│        ↓                                                        │
│                                                                 │
│  3. Response 201 Created                                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           │
                           │
┌──────────────────────────▼──────────────────────────────────────┐
│                   OUTBOX PUBLISHER                              │
│                   (Background Worker)                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Polling Loop (каждые 5 секунд):                               │
│                                                                 │
│  1. SELECT * FROM outbox                                        │
│     WHERE processed_at IS NULL                                  │
│     ORDER BY id ASC                                             │
│     LIMIT 100;                                                  │
│        ↓                                                        │
│                                                                 │
│  2. Для каждого события:                                       │
│     ┌─────────────────────────────────────┐                    │
│     │ Publish to Kafka                    │                    │
│     │   topic: events.asset.created       │                    │
│     │   key: event_id                     │                    │
│     │   value: payload                    │                    │
│     └─────────────────────────────────────┘                    │
│        ↓                                                        │
│                                                                 │
│  3. UPDATE outbox                                               │
│     SET processed_at = NOW()                                    │
│     WHERE id = ?;                                               │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
                           │
                           │
                           ▼
                    ┌─────────────┐
                    │    KAFKA    │
                    │   events.*  │
                    └─────────────┘
```

### Детали реализации

#### 1. Схема БД

```sql
CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    event_id VARCHAR(255) NOT NULL UNIQUE,    -- UUID события
    event_type VARCHAR(255) NOT NULL,          -- тип (MediaCreated, etc)
    aggregate_id VARCHAR(255) NOT NULL,        -- ID сущности
    payload JSONB NOT NULL,                    -- полное событие в JSON
    occurred_at TIMESTAMP NOT NULL,            -- когда произошло
    processed_at TIMESTAMP NULL,               -- когда опубликовано
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_outbox_pending ON outbox(processed_at) 
    WHERE processed_at IS NULL;
```

#### 2. Сохранение события

```go
func (s *Service) CreateMedia(ctx context.Context, req CreateMediaRequest) error {
    // Начинаем транзакцию
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 1. Создаём медиа
    media := Media{
        ID:     uuid.New().String(),
        Status: "created",
        // ...
    }
    
    if err := s.repo.SaveInTx(ctx, tx, media); err != nil {
        return err
    }
    
    // 2. Создаём событие
    event := MediaCreatedEvent{
        EventID:     uuid.New().String(),
        EventType:   "MediaCreated",
        AggregateID: media.ID,
        OccurredAt:  time.Now(),
        Payload: MediaCreatedPayload{
            MediaID: media.ID,
            Type:    media.Type,
            // ...
        },
    }
    
    // 3. Сохраняем в outbox (в той же транзакции!)
    if err := s.outboxRepo.AddInTx(ctx, tx, event); err != nil {
        return err
    }
    
    // ✅ Атомарный COMMIT - либо всё, либо ничего
    return tx.Commit()
}
```

#### 3. Публикация событий

```go
type Publisher struct {
    outboxRepo *OutboxRepo
    producer   *kafka.Producer
    interval   time.Duration
    batchSize  int
    logger     zerolog.Logger
}

func (p *Publisher) Start(ctx context.Context) error {
    ticker := time.NewTicker(p.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
            
        case <-ticker.C:
            // Читаем batch непубликованных событий
            records, err := p.outboxRepo.GetPending(ctx, p.batchSize)
            if err != nil {
                p.logger.Error().Err(err).Msg("failed to get pending")
                continue
            }
            
            // Публикуем каждое
            for _, record := range records {
                // Publish в Kafka
                err := p.producer.Publish(ctx, record.EventID, record.Payload)
                if err != nil {
                    p.logger.Error().
                        Str("event_id", record.EventID).
                        Err(err).
                        Msg("failed to publish")
                    continue
                }
                
                // Помечаем как обработанное
                _ = p.outboxRepo.MarkProcessed(ctx, record.ID)
            }
        }
    }
}
```

---

## Архитектура в нашем проекте

### Компоненты

```
internal/
├── media/
│   ├── outbox/
│   │   └── publisher.go         # Outbox Publisher
│   ├── service/
│   │   └── service.go           # Бизнес-логика + сохранение в outbox
│   └── repository/
│       └── repository.go        # Работа с БД
└── storage/
    └── postgres/
        ├── outbox_repo.go       # Репозиторий для outbox таблицы
        └── media_repo.go        # Репозиторий для медиа
```

### Поток данных

```
User Request
    ↓
[HTTP Handler] → [Service] → [Repository + OutboxRepo]
                                    ↓
                            [Postgres Transaction]
                            ├── media table
                            └── outbox table
                                    ↓
                            [COMMIT] ✅
    
[Background]
    ↓
[OutboxPublisher] → polls outbox table
    ↓
[Kafka Producer] → publishes events
    ↓
[Update outbox] → marks as processed
```

### Конфигурация Publisher

```go
// cmd/media/main.go
func main() {
    // ...
    
    publisher, err := outbox.NewPublisher(outbox.PublisherConfig{
        OutboxRepo: outboxRepo,
        Producer:   kafkaProducer,
        Interval:   5 * time.Second,  // как часто проверять outbox
        BatchSize:  100,               // сколько событий за раз
        Logger:     logger,
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Запускаем в отдельной горутине
    go func() {
        if err := publisher.Start(ctx); err != nil {
            logger.Error().Err(err).Msg("publisher stopped")
        }
    }()
    
    // ...
}
```

---

## Гарантии и ограничения

### ✅ Что гарантирует Outbox Pattern

1. **At-least-once delivery**
    - Каждое событие будет опубликовано хотя бы один раз
    - Может быть опубликовано повторно при сбоях

2. **Ordering в рамках одного aggregate**
    - События для одной сущности публикуются в порядке их создания
    - Используется `ORDER BY id ASC` в запросе

3. **Durability**
    - События сохраняются в БД и переживают рестарты сервиса
    - Нет потери событий даже при падении Kafka

### ⚠️ Ограничения

1. **НЕ exactly-once**
    - При сбое между публикацией и маркировкой события может быть отправлено повторно
    - **Решение**: Consumer должен быть идемпотентным

2. **НЕ строгий ordering между разными aggregates**
    - События для разных сущностей могут быть опубликованы не в том порядке
    - **Решение**: Если нужен strict ordering — используйте Saga Pattern

3. **Задержка публикации**
    - События публикуются не мгновенно, а с задержкой = `interval`
    - **Решение**: Уменьшите interval (но не слишком, иначе создадите нагрузку на БД)

4. **Нагрузка на БД**
    - Polling создаёт дополнительные запросы к БД
    - **Решение**: Оптимизируйте interval и batch size

### 🎯 Best Practices

#### 1. Идемпотентность Consumer

```go
// Consumer должен проверять, не обрабатывал ли он уже это событие
func (c *Consumer) HandleEvent(ctx context.Context, event Event) error {
    // Проверяем в Redis/БД
    exists, err := c.dedup.Exists(ctx, event.EventID)
    if err != nil {
        return err
    }
    
    if exists {
        c.logger.Warn().
            Str("event_id", event.EventID).
            Msg("duplicate event, skipping")
        return nil  // уже обработали
    }
    
    // Обрабатываем
    if err := c.processEvent(ctx, event); err != nil {
        return err
    }
    
    // Сохраняем event_id для дедупликации
    return c.dedup.Save(ctx, event.EventID, 24*time.Hour)
}
```

#### 2. Мониторинг Lag

```go
// Периодически проверяем отставание
func (p *Publisher) MonitorLag(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            count, err := p.outboxRepo.GetPendingCount(ctx)
            if err != nil {
                p.logger.Error().Err(err).Msg("failed to get lag")
                continue
            }
            
            p.logger.Info().
                Int64("pending_events", count).
                Msg("outbox lag")
            
            // Отправляем метрику в Prometheus
            outboxLagGauge.Set(float64(count))
        }
    }
}
```

#### 3. Cleanup старых событий

```sql
-- Удаляем обработанные события старше 7 дней
DELETE FROM outbox 
WHERE processed_at IS NOT NULL 
  AND processed_at < NOW() - INTERVAL '7 days';
```

```go
func (r *OutboxRepo) CleanupProcessed(ctx context.Context, olderThan time.Duration) error {
    query := `
        DELETE FROM outbox 
        WHERE processed_at IS NOT NULL 
          AND processed_at < $1
    `
    _, err := r.db.ExecContext(ctx, query, time.Now().Add(-olderThan))
    return err
}
```

---

## Примеры использования

### Пример 1: Create Media

```go
func (s *MediaService) CreateMedia(ctx context.Context, req CreateMediaRequest) (*Media, error) {
    // Валидация
    if err := req.Validate(); err != nil {
        return nil, err
    }
    
    // Транзакция
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback()
    
    // Создаём медиа
    media := &Media{
        ID:        uuid.New().String(),
        Type:      req.Type,
        Source:    req.Source,
        Status:    StatusCreated,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }
    
    // Сохраняем в БД
    if err := s.mediaRepo.CreateInTx(ctx, tx, media); err != nil {
        return nil, fmt.Errorf("create media: %w", err)
    }
    
    // Создаём событие
    event := events.MediaCreated{
        BaseEvent: events.BaseEvent{
            EventID:     uuid.New().String(),
            EventType:   "MediaCreated",
            AggregateID: media.ID,
            OccurredAt:  time.Now(),
        },
        MediaID: media.ID,
        Type:    media.Type,
        Source:  media.Source,
    }
    
    // Сохраняем в outbox
    if err := s.outboxRepo.AddInTx(ctx, tx, event); err != nil {
        return nil, fmt.Errorf("add to outbox: %w", err)
    }
    
    // Коммитим всё атомарно
    if err := tx.Commit(); err != nil {
        return nil, fmt.Errorf("commit tx: %w", err)
    }
    
    s.logger.Info().
        Str("media_id", media.ID).
        Str("event_id", event.EventID).
        Msg("media created with outbox event")
    
    return media, nil
}
```

### Пример 2: Update Status

```go
func (s *MediaService) UpdateStatus(ctx context.Context, id string, newStatus Status) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Получаем текущую медиа
    media, err := s.mediaRepo.GetByIDInTx(ctx, tx, id)
    if err != nil {
        return err
    }
    
    oldStatus := media.Status
    
    // Обновляем статус
    media.Status = newStatus
    media.UpdatedAt = time.Now()
    
    if err := s.mediaRepo.UpdateInTx(ctx, tx, media); err != nil {
        return err
    }
    
    // Создаём событие об изменении статуса
    event := events.MediaStatusChanged{
        BaseEvent: events.BaseEvent{
            EventID:     uuid.New().String(),
            EventType:   "MediaStatusChanged",
            AggregateID: id,
            OccurredAt:  time.Now(),
        },
        MediaID:   id,
        OldStatus: oldStatus,
        NewStatus: newStatus,
    }
    
    if err := s.outboxRepo.AddInTx(ctx, tx, event); err != nil {
        return err
    }
    
    return tx.Commit()
}
```

---

## Мониторинг и Troubleshooting

### Метрики для мониторинга

```go
// Prometheus metrics
var (
    outboxLag = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "outbox_pending_events",
        Help: "Number of pending events in outbox",
    })
    
    outboxPublished = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "outbox_events_published_total",
        Help: "Total number of events published",
    }, []string{"event_type"})
    
    outboxErrors = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "outbox_publish_errors_total",
        Help: "Total number of publish errors",
    }, []string{"event_type", "error_type"})
    
    outboxPublishDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
        Name: "outbox_publish_duration_seconds",
        Help: "Time taken to publish events",
    }, []string{"event_type"})
)
```

### Логирование с Zerolog

```go
logger.Info().
    Str("event_id", event.EventID).
    Str("event_type", event.EventType).
    Str("aggregate_id", event.AggregateID).
    Int64("outbox_id", record.ID).
    Dur("latency", time.Since(record.OccurredAt)).
    Msg("event published")
```

### Типичные проблемы и решения

#### 1. События не публикуются

**Симптомы:**
- Outbox таблица растёт
- В Kafka нет новых событий

**Диагностика:**
```sql
-- Проверяем количество pending событий
SELECT COUNT(*) FROM outbox WHERE processed_at IS NULL;

-- Смотрим самые старые события
SELECT * FROM outbox 
WHERE processed_at IS NULL 
ORDER BY id ASC 
LIMIT 10;
```

**Возможные причины:**
- Publisher не запущен
- Kafka недоступен
- Ошибки в логах Publisher

#### 2. Большая задержка (lag)

**Симптомы:**
- События публикуются с большой задержкой
- Lag растёт

**Решения:**
```go
// Увеличить batch size
publisher.batchSize = 500

// Уменьшить interval
publisher.interval = 1 * time.Second

// Запустить несколько Publisher (требует координации)
```

#### 3. Дублирование событий

**Симптомы:**
- Одно и то же событие обработано несколько раз

**Причина:**
- Сбой между публикацией и маркировкой

**Решение:**
```go
// Consumer ОБЯЗАН быть идемпотентным
func (c *Consumer) HandleEvent(ctx context.Context, event Event) error {
    // Дедупликация по event_id
    if c.alreadyProcessed(event.EventID) {
        return nil
    }
    
    // ... обработка
    
    c.markProcessed(event.EventID)
    return nil
}
```

---

## Заключение

Outbox Pattern — это **production-ready решение** для надёжной публикации событий в микросервисной архитектуре.

### Ключевые моменты:

✅ Используйте для критичных событий  
✅ Всегда делайте Consumer идемпотентным  
✅ Мониторьте lag и ошибки  
✅ Настраивайте interval и batch size под вашу нагрузку  
✅ Не забывайте cleanup старых событий

### Дальнейшее развитие:

1. **CDC (Change Data Capture)** — Debezium вместо polling
2. **Partitioning** — горизонтальное масштабирование
3. **Priority queues** — разные интервалы для разных типов событий

---

**Ссылки:**
- [Microservices Patterns (Chris Richardson)](https://microservices.io/patterns/data/transactional-outbox.html)
- [Implementing the Outbox Pattern](https://debezium.io/blog/2019/02/19/reliable-microservices-data-exchange-with-the-outbox-pattern/)