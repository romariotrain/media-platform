# Publish Service

Публикация обработанных медиафайлов. Копирует файлы в публичную директорию и формирует URL.

## Порт

`:8083`

## Процесс публикации

1. Получает команду `StartPublish` с `source_paths` (пути к обработанным файлам)
2. Копирует файлы в `published/<asset_id>/`
3. Формирует публичные URL на основе `PUBLISH_BASE_URL`
4. Отправляет событие `PublishSucceeded` с `public_urls`

## HTTP API

| Метод | Endpoint | Описание |
|-------|----------|----------|
| `GET` | `/publications?id=<uuid>` | Публикация по ID |
| `GET` | `/publications/list?asset_id=<uuid>` | Список публикаций по asset |
| `GET` | `/health` | Health check |

## Kafka

**Слушает:**
- `commands.publish.start` — команда опубликовать файлы

**Отправляет:**
- `events.publish.succeeded` — публикация завершена (с public_urls)
- `events.publish.failed` — ошибка публикации

## Хранилище

Файлы копируются в `PUBLISH_DIR` (по умолчанию `published/`).
URL формируется как `PUBLISH_BASE_URL/<asset_id>/<file>`.

## БД таблицы

- `publications` — записи публикаций (source_paths, published_paths, public_urls)
- `publish_outbox` — outbox событий

## Ключевые файлы

- `storage/fs/file_publisher.go` — копирование файлов + генерация URL
- `service/publish.go` — бизнес-логика публикации
- `httpapi/handlers.go` — HTTP API для чтения публикаций
