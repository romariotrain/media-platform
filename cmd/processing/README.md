# Processing Service

Обработка медиафайлов через FFmpeg. Фоновый worker без HTTP API.

## Порт

Нет HTTP API. Работает только через Kafka.

## Обработка

При получении команды `StartProcessing`:
1. Создаёт задачу в БД со статусом `processing`
2. Запускает FFmpeg:
   - Транскод в 720p (`-vf scale=-2:720`)
   - Транскод в 1080p (`-vf scale=-2:1080`)
   - Генерация thumbnail (кадр из середины видео)
   - Извлечение метаданных через `ffprobe` (duration, codec, resolution, bitrate)
3. Обновляет задачу с результатами
4. Отправляет событие `ProcessingSucceeded` или `ProcessingFailed`

Выходные файлы: `output/<task_id>/video_720p.mp4`, `video_1080p.mp4`, `thumbnail.jpg`.

## Kafka

**Слушает:**
- `commands.processing.start` — команда обработать файл

**Отправляет:**
- `events.processing.succeeded` — обработка завершена (с output_paths и metadata)
- `events.processing.failed` — ошибка обработки

## БД таблицы

- `processing_tasks` — задачи обработки (status, input_path, output_paths, metadata)
- `processing_outbox` — outbox событий

## Ключевые файлы

- `ffmpeg/processor.go` — обёртка над FFmpeg/ffprobe
- `service/process.go` — бизнес-логика обработки
- `kafka/consumer.go` — приём команд
