package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/romariotrain/media-platform/internal/ingest/models"
)

// UploadFile обрабатывает загрузку файла по токену.
// Валидирует токен, проверяет файл, сохраняет на диск и публикует событие.
func (s *Service) UploadFile(ctx context.Context, token string, fileName string, fileSize int64, contentType string, body io.Reader) (*models.UploadSession, error) {
	// 1. Валидация входных данных
	if token == "" || fileName == "" {
		return nil, models.ErrInvalidArgument
	}

	// 2. Поиск сессии по токену
	session, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("get session by token: %w", err)
	}

	// 3. Проверка срока действия токена
	if s.clock().After(session.ExpiresAt) {
		return nil, models.ErrTokenExpired
	}

	// 4. Проверка статуса (должен быть "ready")
	if session.Status != models.StatusReady {
		return nil, models.ErrTokenUsed
	}

	// 5. Валидация размера файла
	if fileSize > MaxFileSize {
		return nil, models.ErrFileTooLarge
	}

	// 6. Валидация content type
	if !AllowedContentTypes[contentType] {
		return nil, models.ErrUnsupportedType
	}

	// 7. Сохраняем файл на диск
	subDir := session.AssetID.String()
	ext := filepath.Ext(fileName)
	storedFileName := session.ID.String() + ext

	storagePath, err := s.fileStore.Save(subDir, storedFileName, body)
	if err != nil {
		return nil, fmt.Errorf("save file: %w", err)
	}

	// 8. BEGIN TX: обновляем сессию + добавляем событие в outbox
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		s.fileStore.Delete(storagePath)
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := s.repo.UpdateUploadInfoTx(ctx, tx, session.ID, fileName, fileSize, contentType, storagePath); err != nil {
		s.fileStore.Delete(storagePath)
		return nil, fmt.Errorf("update upload info: %w", err)
	}

	// Обновляем локальную копию для события
	session.Status = models.StatusUploaded
	session.FileName = &fileName
	session.FileSize = &fileSize
	session.ContentType = &contentType
	session.StoragePath = &storagePath

	event := models.NewIngestUploadedEvent(session.SagaID, session)
	if err := s.outboxRepo.Add(ctx, tx, string(models.EventIngestUploaded), session.ID.String(), session.SagaID, event.Payload); err != nil {
		s.fileStore.Delete(storagePath)
		return nil, fmt.Errorf("add event to outbox: %w", err)
	}

	// COMMIT
	if err := tx.Commit(); err != nil {
		s.fileStore.Delete(storagePath)
		return nil, fmt.Errorf("commit tx: %w", err)
	}

	return session, nil
}

// GetUploadToken возвращает информацию о сессии по токену
func (s *Service) GetUploadToken(ctx context.Context, token string) (*models.UploadSession, error) {
	if token == "" {
		return nil, models.ErrInvalidArgument
	}

	session, err := s.repo.GetByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if s.clock().After(session.ExpiresAt) {
		return nil, models.ErrTokenExpired
	}

	return session, nil
}
