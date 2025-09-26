package utils

import (
	"fmt"
	"log/slog"

	"os"
	"os/exec"
	"path/filepath"
)

func ConvertToMP4(inputPath string, outputPath string) error {
	if filepath.Clean(inputPath) == filepath.Clean(outputPath) {
		return fmt.Errorf("входной и выходной пути совпадают: %s", inputPath)
	}

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		err := fmt.Errorf("входной файл не найден: %s", inputPath)
		slog.Error(err.Error(), "path", inputPath)
		return err
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg не установлен или не в PATH: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию: %w", err)
	}

	args := []string{
		"-i", inputPath,
		"-c:v", "libx264",
		"-crf", "26",
		"-preset", "ultrafast",
		"-c:a", "aac",
		"-b:a", "192k",
		"-movflags", "+faststart",
		outputPath,
	}

	slog.Info("Запуск ffmpeg", "args", args)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка при выполнении ffmpeg: %w", err)
	}

	slog.Info("✅ Успешно сконвертировано", "from", inputPath, "to", outputPath)
	return nil
}
