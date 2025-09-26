package utils

import (
	"fmt"
	"log/slog"

	"os"
	"os/exec"
	"path/filepath"
)

// ConvertToMP4 конвертирует видео в веб-совместимый MP4
func ConvertToMP4(inputPath string, outputPath string) error {
	// Проверяем, существует ли входной файл
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		return fmt.Errorf("входной файл не найден: %s", inputPath)
	}

	// Проверяем, установлен ли ffmpeg
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg не установлен или не в PATH: %v", err)
	}

	// Создаем директорию для выходного файла, если нужно
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию: %w", err)
	}

	// Аргументы ffmpeg
	//
	// -i input.avi                   : входной файл
	// -c:v libx264                   : видео в H.264
	// -crf 23                        : качество (18–28), 23 — баланс
	// -preset fast                   : скорость кодирования
	// -c:a aac                       : аудио в AAC
	// -b:a 128k                      : битрейт аудио
	// -movflags +faststart           : оптимизация для стриминга (проигрывание до загрузки)
	// output.mp4                     : выход
	args := []string{
		"-i", inputPath,
		"-c:v", "libx264",
		"-crf", "23",
		"-preset", "fast",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	}

	cmd := exec.Command("ffmpeg", args...)

	// Выводим команду для отладки
	slog.Info(fmt.Sprintf("Запуск: ffmpeg %v\n", args))

	// Запускаем
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ошибка при выполнении ffmpeg: %w", err)
	}

	if err := os.Remove(inputPath); err != nil {
		return fmt.Errorf("не удалось удалить файл с диска: %w", err)
	}	
	slog.Info(fmt.Sprintf("✅ Успешно сконвертировано: %s → %s\n", inputPath, outputPath))
	return nil
}
