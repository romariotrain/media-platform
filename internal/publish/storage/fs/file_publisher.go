package fs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/romariotrain/media-platform/internal/publish/models"
)

// FilePublisher копирует обработанные файлы в публичную директорию
type FilePublisher struct {
	publishDir string
	baseURL    string
}

// NewFilePublisher создаёт новый FilePublisher
func NewFilePublisher(publishDir string, baseURL string) (*FilePublisher, error) {
	if err := os.MkdirAll(publishDir, 0o755); err != nil {
		return nil, fmt.Errorf("create publish dir: %w", err)
	}
	return &FilePublisher{
		publishDir: publishDir,
		baseURL:    baseURL,
	}, nil
}

// PublishResult содержит результаты публикации
type PublishResult struct {
	PublishedPaths *models.PublishedPaths
	PublicURLs     *models.PublicURLs
}

// Publish копирует файлы из source в published директорию и формирует публичные URL
func (p *FilePublisher) Publish(assetID string, sources *models.SourcePaths) (*PublishResult, error) {
	assetDir := filepath.Join(p.publishDir, assetID)
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		return nil, fmt.Errorf("create asset dir: %w", err)
	}

	published := &models.PublishedPaths{}
	urls := &models.PublicURLs{}

	if sources.Video720p != "" {
		dst := filepath.Join(assetDir, "video_720p.mp4")
		if err := copyFile(sources.Video720p, dst); err != nil {
			return nil, fmt.Errorf("copy video 720p: %w", err)
		}
		published.Video720p = dst
		urls.Video720p = p.baseURL + "/" + assetID + "/video_720p.mp4"
	}

	if sources.Video1080p != "" {
		dst := filepath.Join(assetDir, "video_1080p.mp4")
		if err := copyFile(sources.Video1080p, dst); err != nil {
			return nil, fmt.Errorf("copy video 1080p: %w", err)
		}
		published.Video1080p = dst
		urls.Video1080p = p.baseURL + "/" + assetID + "/video_1080p.mp4"
	}

	if sources.Thumbnail != "" {
		dst := filepath.Join(assetDir, "thumbnail.jpg")
		if err := copyFile(sources.Thumbnail, dst); err != nil {
			return nil, fmt.Errorf("copy thumbnail: %w", err)
		}
		published.Thumbnail = dst
		urls.Thumbnail = p.baseURL + "/" + assetID + "/thumbnail.jpg"
	}

	return &PublishResult{
		PublishedPaths: published,
		PublicURLs:     urls,
	}, nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	return dstFile.Sync()
}
