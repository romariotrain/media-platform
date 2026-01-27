# 🎉 Kafka Producer — Production-Ready Implementation

## ✅ Что сделано

Создана production-ready реализация Kafka Producer с:

### 1. 🔄 Retry с Exponential Backoff
- Автоматические retry при временных ошибках
- Exponential backoff: 100ms → 200ms → 400ms → 800ms (cap at 5s)
- Умное определение retriable/non-retriable ошибок
- Context cancellation support

### 2. 📝 Structured Logging (zerolog)
- Детальные логи всех операций
- JSON для production, pretty console для development
- Контекст для каждого сообщения (key, size, duration, attempts)
- Разные уровни (debug, info, warn, error)

### 3. 📊 Метрики для мониторинга
- `MessagesPublished` — успешно опубликованные
- `MessagesFailed` — проваленные
- `RetriesTotal` — общее количество retry
- `AvgPublishTime` — среднее время публикации

### 4. ✅ Валидация конфигурации
- Проверка при создании Producer
- Понятные сообщения об ошибках
- Defaults для всех параметров

### 5. 🛑 Graceful Shutdown
- Корректное закрытие с flush pending messages
- Timeout 30 секунд на завершение операций
- Финальные метрики в логах

### 6. ❤️ Health Check
- Проверка работоспособности Producer
- Анализ error rate
- Готово для Kubernetes probes

### 7. 📦 Batch Publishing
- Эффективная публикация нескольких сообщений
- Атомарная операция (all or nothing)
- Retry для всего batch

### 8. 🧪 Тесты
- 20+ unit-тестов
- Покрытие всех сценариев
- Benchmark для производительности

---

## 📁 Файлы

```
outputs/
├── producer_improved.go       # ✨ Улучшенная реализация Producer
├── producer_test.go           # ✨ Unit-тесты (20+ тестов)
├── docs/
│   ├── KAFKA_PRODUCER.md      # 📖 Полная документация
│   ├── KAFKA_QUICK_START.md   # 🚀 Quick start guide
│   └── ...                    # Другие документы
└── README_KAFKA.md            # 👋 Этот файл
```

---

## 🎯 Как читать

### Для быстрого старта:
1. **Этот файл** — обзор улучшений
2. `docs/KAFKA_QUICK_START.md` — пошаговое применение

### Для понимания деталей:
1. `docs/KAFKA_PRODUCER.md` — полная документация
    - Retry логика
    - Метрики
    - Best practices
    - Troubleshooting

### Для изучения кода:
1. `producer_improved.go` — улучшенная реализация
2. `producer_test.go` — примеры тестов

---

## 🚀 Быстрый старт

### 1. Установите zerolog (если ещё не установлен)

```bash
go get github.com/rs/zerolog
```

### 2. Замените producer.go

```bash
cd /path/to/media-platform

# Backup
cp internal/media/kafka/producer.go internal/media/kafka/producer_old.go

# Замените
cp producer_improved.go internal/media/kafka/producer.go
```

### 3. Обновите создание Producer

**Было:**
```go
producer := kafka.NewProducer([]string{"localhost:9092"}, "events")
```

**Стало:**
```go
logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

producer, err := kafka.NewProducer(kafka.ProducerConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "events.media.created",
    Logger:  logger,
})
if err != nil {
    log.Fatal(err)
}
defer producer.Close()
```

**Детальная инструкция:** `docs/KAFKA_QUICK_START.md`

---

## 📊 Сравнение: До и После

### Код

**До (простой, но хрупкий):**
```go
func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
    err := p.writer.WriteMessages(ctx, kafkago.Message{
        Key:   []byte(key),
        Value: value,
    })
    if err != nil {
        return fmt.Errorf("kafka publish: %w", err)
    }
    return nil
}
```

**После (production-ready):**
```go
func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
    // ✅ Проверка closed
    // ✅ Structured logging
    // ✅ Retry loop с exponential backoff
    // ✅ Retriable/non-retriable error detection
    // ✅ Metrics tracking
    // ✅ Context cancellation support
    // 350+ строк надёжного кода
}
```

### Логи

**До:**
```
(ничего)
```

**После (JSON):**
```json
{
  "level": "info",
  "component": "kafka_producer",
  "topic": "events.media.created",
  "brokers": ["localhost:9092"],
  "max_retries": 3,
  "time": "2026-01-18T14:00:00Z",
  "message": "kafka producer created"
}

{
  "level": "warn",
  "component": "kafka_producer",
  "attempt": 2,
  "backoff": 100000000,
  "error": "connection refused",
  "message": "retrying publish"
}

{
  "level": "info",
  "component": "kafka_producer",
  "duration": 150000000,
  "attempts": 2,
  "message": "message published successfully"
}
```

### Поведение при ошибках

**До:**
```
Error: connection refused
(fails immediately, event lost)
```

**После:**
```
Attempt 1: connection refused
Wait 100ms...
Attempt 2: connection refused
Wait 200ms...
Attempt 3: connection refused
Wait 400ms...
Attempt 4: SUCCESS! ✅
```

### Метрики

**До:**
```
(ничего)
```

**После:**
```go
metrics := producer.GetMetrics()
// MessagesPublished: 10000
// MessagesFailed: 50
// RetriesTotal: 200
// AvgPublishTime: 45ms

// Можно отправить в Prometheus
kafkaMessagesPublished.Set(float64(metrics.MessagesPublished))
```

---

## 🎓 Ключевые концепции

### Retry Strategy

```
Error → Retriable? → No → Return error immediately
         ↓ Yes
    Exponential backoff
         ↓
    Retry (up to MaxRetries times)
```

**Retriable errors:**
- Connection refused/reset
- Timeout
- Leader not available
- Temporary failures

**Non-retriable errors:**
- Invalid message
- Message too large
- Authorization failed
- Context cancelled

### Exponential Backoff

```
Attempt 1: immediate (0ms)
Attempt 2: 100ms  (baseBackoff * 2^0)
Attempt 3: 200ms  (baseBackoff * 2^1)
Attempt 4: 400ms  (baseBackoff * 2^2)
Attempt 5: 800ms  (baseBackoff * 2^3)
Attempt 6: 1600ms (baseBackoff * 2^4)
...
Max: 5000ms (capped)
```

### Метрики в реальном времени

```go
type ProducerMetrics struct {
    MessagesPublished atomic.Int64 // thread-safe
    MessagesFailed    atomic.Int64
    RetriesTotal      atomic.Int64
    PublishDuration   atomic.Int64
}

// Любая горутина может безопасно читать/писать
metrics.MessagesPublished.Add(1)
count := metrics.MessagesPublished.Load()
```

---

## 💡 Примеры использования

### Базовое

```go
logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

producer, err := kafka.NewProducer(kafka.ProducerConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "events",
    Logger:  logger,
})
if err != nil {
    log.Fatal(err)
}
defer producer.Close()

ctx := context.Background()
err = producer.Publish(ctx, "key", []byte("value"))
```

### С custom конфигурацией

```go
producer, err := kafka.NewProducer(kafka.ProducerConfig{
    Brokers:      []string{"kafka1:9092", "kafka2:9092"},
    Topic:        "events",
    MaxRetries:   5,                      // больше retry
    RetryBackoff: 200 * time.Millisecond, // больше backoff
    WriteTimeout: 5 * time.Second,
    BatchSize:    50,
    Async:        true,
    Logger:       logger,
})
```

### С timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := producer.Publish(ctx, "key", value)
if errors.Is(err, context.DeadlineExceeded) {
    log.Println("Publish timed out")
}
```

### Batch publishing

```go
messages := []kafka.Message{
    {Key: "event-1", Value: []byte(`{"id":"1"}`)},
    {Key: "event-2", Value: []byte(`{"id":"2"}`)},
    {Key: "event-3", Value: []byte(`{"id":"3"}`)},
}

err := producer.PublishBatch(ctx, messages)
```

### Мониторинг метрик

```go
ticker := time.NewTicker(1 * time.Minute)
defer ticker.Stop()

for range ticker.C {
    metrics := producer.GetMetrics()
    
    logger.Info().
        Int64("published", metrics.MessagesPublished).
        Int64("failed", metrics.MessagesFailed).
        Int64("retries", metrics.RetriesTotal).
        Dur("avg_time", metrics.AvgPublishTime).
        Msg("kafka metrics")
    
    // Alert если error rate > 10%
    if metrics.MessagesPublished > 0 {
        errorRate := float64(metrics.MessagesFailed) / float64(metrics.MessagesPublished)
        if errorRate > 0.1 {
            alerting.Send("High Kafka error rate: %.2f%%", errorRate*100)
        }
    }
}
```

### Health check

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
    defer cancel()
    
    if err := producer.HealthCheck(ctx); err != nil {
        http.Error(w, err.Error(), http.StatusServiceUnavailable)
        return
    }
    
    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}
```

---

## 🧪 Тестирование

### Запуск тестов

```bash
# Скопируйте тесты
cp producer_test.go internal/media/kafka/

# Запустите
go test ./internal/media/kafka/...

# С verbose
go test -v ./internal/media/kafka/...

# С покрытием
go test -cover ./internal/media/kafka/...

# Benchmark
go test -bench=. ./internal/media/kafka/...
```

### Покрытие

- ✅ Создание Producer (успех + валидация)
- ✅ Defaults и custom config
- ✅ Retriable/non-retriable errors
- ✅ Метрики (published, failed, retries, avg time)
- ✅ Close и double-close
- ✅ Publish after close
- ✅ Batch publishing
- ✅ Health check
- ✅ Context cancellation

**Всего:** 20+ тестов + 1 benchmark

---

## 🎯 Best Practices

### 1. Context timeout

```go
// ✅ Good
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
err := producer.Publish(ctx, key, value)

// ❌ Bad
err := producer.Publish(context.Background(), key, value)
```

### 2. Graceful shutdown

```go
defer func() {
    logger.Info().Msg("closing kafka producer")
    if err := producer.Close(); err != nil {
        logger.Error().Err(err).Msg("error closing producer")
    }
}()
```

### 3. Мониторинг

```go
// Периодически проверяем метрики
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        metrics := producer.GetMetrics()
        // Check error rate, avg time, etc.
    }
}()
```

### 4. Настройка retry

```go
// Критичные события
producer, _ := kafka.NewProducer(kafka.ProducerConfig{
    MaxRetries:   10,
    RetryBackoff: 500 * time.Millisecond,
})

// Некритичные
producer, _ := kafka.NewProducer(kafka.ProducerConfig{
    MaxRetries:   1,
    RetryBackoff: 50 * time.Millisecond,
})
```

---

## 🐛 Troubleshooting

### Все сообщения fail

**Симптомы:**
```
MessagesFailed: 1000
MessagesPublished: 0
```

**Проверка:**
```bash
# Kafka работает?
docker ps | grep kafka

# Connectivity?
telnet localhost 9092
```

**Решение:**
```bash
docker compose -f deploy/docker-compose.yml up -d kafka
```

### Много retry

**Симптомы:**
```
RetriesTotal: 5000
MessagesPublished: 1000
```

**Причины:** Kafka overloaded, network issues

**Решение:**
```go
// Используйте batch
err := producer.PublishBatch(ctx, messages)

// Увеличьте backoff
producer, _ := kafka.NewProducer(kafka.ProducerConfig{
    RetryBackoff: 500 * time.Millisecond,
})
```

### Медленная публикация

**Симптомы:**
```
AvgPublishTime: 2s
```

**Решение:**
```go
// Async mode
producer, _ := kafka.NewProducer(kafka.ProducerConfig{
    Async: true,
})

// Batch
err := producer.PublishBatch(ctx, messages)
```

**Детальный troubleshooting:** `docs/KAFKA_PRODUCER.md`

---

## 📚 Документация

### Файлы

1. **README_KAFKA.md** (этот файл) — обзор улучшений
2. **docs/KAFKA_QUICK_START.md** — пошаговое применение
3. **docs/KAFKA_PRODUCER.md** — полная документация

### В коде

- `producer_improved.go` — 500+ строк production-ready кода
- `producer_test.go` — 20+ тестов

---

## ✅ Чек-лист

- [ ] Прочитал этот README
- [ ] Прочитал KAFKA_QUICK_START.md
- [ ] Установил zerolog
- [ ] Заменил producer.go
- [ ] Обновил код создания Producer
- [ ] Добавил graceful shutdown
- [ ] Запустил тесты
- [ ] Проверил структурированные логи
- [ ] Настроил мониторинг метрик
- [ ] Добавил health check

---

## 🎁 Бонус: Интеграция с Outbox Publisher

Улучшенный Kafka Producer идеально работает с Outbox Publisher:

```go
// Создаём Kafka Producer
kafkaProducer, err := kafka.NewProducer(kafka.ProducerConfig{
    Brokers: []string{"localhost:9092"},
    Topic:   "events.media.created",
    Logger:  logger,
})

// Используем в Outbox Publisher
outboxPublisher, err := outbox.NewPublisher(outbox.PublisherConfig{
    OutboxRepo: outboxRepo,
    Producer:   kafkaProducer, // наш улучшенный Producer!
    Interval:   5 * time.Second,
    BatchSize:  100,
    Logger:     logger,
})
```

**Преимущества:**
- ✅ Retry логика для событий из Outbox
- ✅ Детальные логи публикации
- ✅ Метрики для Kafka operations
- ✅ Graceful shutdown для обоих компонентов

---

## 🚀 Итого

Вы получили:

1. ✅ **Production-ready Kafka Producer** с retry, logging, metrics
2. ✅ **20+ тестов** — покрытие всех сценариев
3. ✅ **Документацию** — 3 подробных файла
4. ✅ **Примеры** — готовые к использованию snippets
5. ✅ **Best practices** — как использовать правильно

**Всё готово к использованию!** 🚀

Следуй `docs/KAFKA_QUICK_START.md` для применения.

---

**P.S.** Этот Producer можно использовать не только для Outbox Pattern, но и для любой публикации в Kafka. Он универсальный и production-ready! 💪