package ffmpeg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/romariotrain/media-platform/internal/processing/models"
)

// Processor обёртка над FFmpeg для обработки медиафайлов
type Processor struct {
	ffmpegPath  string
	ffprobePath string
	outputDir   string
}

// NewProcessor создаёт новый Processor
func NewProcessor(outputDir string) (*Processor, error) {
	// Проверяем наличие ffmpeg
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("ffmpeg not found in PATH: %w", err)
	}

	// Проверяем наличие ffprobe
	ffprobePath, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("ffprobe not found in PATH: %w", err)
	}

	// Создаём output директорию
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	return &Processor{
		ffmpegPath:  ffmpegPath,
		ffprobePath: ffprobePath,
		outputDir:   outputDir,
	}, nil
}

// ProcessResult содержит результаты обработки
type ProcessResult struct {
	OutputPaths *models.OutputPaths
	Metadata    *models.Metadata
}

// Process обрабатывает медиафайл: транскодирует, создаёт thumbnail, извлекает метаданные
func (p *Processor) Process(ctx context.Context, inputPath string, taskID string) (*ProcessResult, error) {
	// Создаём директорию для выходных файлов
	taskDir := filepath.Join(p.outputDir, taskID)
	if err := os.MkdirAll(taskDir, 0755); err != nil {
		return nil, fmt.Errorf("create task dir: %w", err)
	}

	result := &ProcessResult{
		OutputPaths: &models.OutputPaths{},
	}

	// 1. Извлекаем метаданные
	metadata, err := p.ExtractMetadata(ctx, inputPath)
	if err != nil {
		return nil, fmt.Errorf("extract metadata: %w", err)
	}
	result.Metadata = metadata

	// 2. Транскодируем в 720p
	output720p := filepath.Join(taskDir, "video_720p.mp4")
	if err := p.Transcode(ctx, inputPath, output720p, 720); err != nil {
		return nil, fmt.Errorf("transcode 720p: %w", err)
	}
	result.OutputPaths.Video720p = output720p

	// 3. Транскодируем в 1080p (если исходник >= 1080p)
	if metadata.Height >= 1080 {
		output1080p := filepath.Join(taskDir, "video_1080p.mp4")
		if err := p.Transcode(ctx, inputPath, output1080p, 1080); err != nil {
			return nil, fmt.Errorf("transcode 1080p: %w", err)
		}
		result.OutputPaths.Video1080p = output1080p
	}

	// 4. Генерируем thumbnail
	thumbnailPath := filepath.Join(taskDir, "thumbnail.jpg")
	if err := p.GenerateThumbnail(ctx, inputPath, thumbnailPath); err != nil {
		return nil, fmt.Errorf("generate thumbnail: %w", err)
	}
	result.OutputPaths.Thumbnail = thumbnailPath

	return result, nil
}

// Transcode транскодирует видео в указанное разрешение
func (p *Processor) Transcode(ctx context.Context, inputPath, outputPath string, height int) error {
	// ffmpeg -i input.mp4 -vf scale=-2:720 -c:v libx264 -crf 23 -c:a aac output_720p.mp4
	args := []string{
		"-i", inputPath,
		"-vf", fmt.Sprintf("scale=-2:%d", height),
		"-c:v", "libx264",
		"-crf", "23",
		"-c:a", "aac",
		"-y", // overwrite
		outputPath,
	}

	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg transcode failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// GenerateThumbnail создаёт превью из видео
func (p *Processor) GenerateThumbnail(ctx context.Context, inputPath, outputPath string) error {
	// ffmpeg -i input.mp4 -ss 00:00:05 -vframes 1 thumbnail.jpg
	args := []string{
		"-i", inputPath,
		"-ss", "00:00:05",
		"-vframes", "1",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg thumbnail failed: %w, stderr: %s", err, stderr.String())
	}

	return nil
}

// ffprobeOutput структура для парсинга вывода ffprobe
type ffprobeOutput struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
		Name     string `json:"format_name"`
	} `json:"format"`
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		SampleRate string `json:"sample_rate"`
	} `json:"streams"`
}

// ExtractMetadata извлекает метаданные из медиафайла
func (p *Processor) ExtractMetadata(ctx context.Context, inputPath string) (*models.Metadata, error) {
	// ffprobe -v quiet -print_format json -show_format -show_streams input.mp4
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, p.ffprobePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w, stderr: %s", err, stderr.String())
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(stdout.Bytes(), &probe); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	metadata := &models.Metadata{
		Format: probe.Format.Name,
	}

	// Парсим duration
	if probe.Format.Duration != "" {
		if dur, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			metadata.Duration = dur
		}
	}

	// Парсим bitrate
	if probe.Format.BitRate != "" {
		if br, err := strconv.ParseInt(probe.Format.BitRate, 10, 64); err == nil {
			metadata.Bitrate = br
		}
	}

	// Ищем видео и аудио потоки
	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			metadata.VideoCodec = stream.CodecName
			metadata.Width = stream.Width
			metadata.Height = stream.Height
		case "audio":
			metadata.AudioCodec = stream.CodecName
		}
	}

	return metadata, nil
}

// IsVideoFile проверяет, является ли файл видео по расширению
func IsVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	videoExts := map[string]bool{
		".mp4":  true,
		".webm": true,
		".mkv":  true,
		".avi":  true,
		".mov":  true,
		".flv":  true,
	}
	return videoExts[ext]
}
