# Orchestrator Service

Координатор Saga-пайплайна. Управляет жизненным циклом медиафайла от загрузки до публикации.

## Порт

`:8084`

## Saga State Machine

```
PREPARE_INGEST → WAIT_UPLOAD → PROCESS_MEDIA → PUBLISH_MEDIA → DONE
```

Каждый шаг:
1. Отправить команду сервису-исполнителю (через outbox → Kafka)
2. Дождаться события-ответа (через Kafka consumer)
3. Обновить состояние саги в БД
4. Перейти к следующему шагу

Если любой шаг возвращает ошибку — сага переходит в `failed`.

## HTTP API

| Метод | Endpoint | Описание |
|-------|----------|----------|
| `POST` | `/sagas` | Создать новую сагу |
| `GET` | `/sagas?id=<uuid>` | Получить сагу по ID |
| `GET` | `/sagas/list?user_id=<string>` | Список саг пользователя |
| `GET` | `/health` | Health check |

### POST /sagas

```json
{
  "asset_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user1"
}
```

Response (`201`):
```json
{
  "id": "...",
  "asset_id": "...",
  "user_id": "user1",
  "status": "running",
  "current_step": "PREPARE_INGEST"
}
```

## Kafka

**Слушает** (events от других сервисов):
- `events.ingest.ready`
- `events.ingest.uploaded`
- `events.processing.succeeded`
- `events.processing.failed`
- `events.publish.succeeded`
- `events.publish.failed`

**Отправляет** (commands через outbox):
- `commands.ingest.prepare`
- `commands.processing.start`
- `commands.publish.start`

## БД таблицы

- `sagas` — состояние саг
- `orchestrator_outbox` — outbox команд

## Ключевые файлы

- `service/create.go` — создание саги и первой команды
- `service/handle_event.go` — state machine, обработка событий
- `kafka/consumer.go` — подписка на 6 event-топиков
- `outbox/publisher.go` — маппинг команд → Kafka-топики
