package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileStore реализует локальное хранилище файлов
type FileStore struct {
	baseDir string
}

// NewFileStore создаёт FileStore, гарантируя существование базовой директории
func NewFileStore(baseDir string) (*FileStore, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create upload dir %s: %w", baseDir, err)
	}
	return &FileStore{baseDir: baseDir}, nil
}

// Save сохраняет файл на диск. Возвращает полный путь к сохранённому файлу.
func (f *FileStore) Save(subDir string, fileName string, r io.Reader) (string, error) {
	dir := filepath.Join(f.baseDir, subDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create sub dir %s: %w", dir, err)
	}

	fullPath := filepath.Join(dir, fileName)
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file %s: %w", fullPath, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		os.Remove(fullPath)
		return "", fmt.Errorf("write file %s: %w", fullPath, err)
	}

	return fullPath, nil
}

// Delete удаляет файл с диска
func (f *FileStore) Delete(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file %s: %w", path, err)
	}
	return nil
}
