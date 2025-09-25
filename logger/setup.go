package logger

import (
	"io"
	"log/slog"
	"os"
)

// SetupLogger инициализирует slog с выводом в консоль и файл
func SetupLogger() *slog.Logger {
	// Открываем файл для записи логов
	logFile, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0664)
	if err != nil {
		// Если не получилось — хотя бы пишем в консоль
		slog.Error("Не удалось открыть файл логов", "error", err)
		return slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	// Создаём мульти-райтер: пишем и в файл, и в консоль
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	// Настраиваем JSON-хендлер
	jsonHandler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelDebug, // Уровень логирования (можно передать как параметр)
	})

	// Создаём логгер
	logger := slog.New(jsonHandler)

	// Устанавливаем как глобальный (опционально)
	slog.SetDefault(logger)

	return logger
}