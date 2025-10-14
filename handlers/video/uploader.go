package video

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"video/config"
	database "video/database"
	"video/utils"

	"log/slog" // <-- добавлен
)


// Метод загрузки видео на сервер
// post?video
func Upload(videoStorage database.VideoStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Получаем файл из формы
		file, handler, err := r.FormFile("video")
		if err != nil {
			slog.Error("Ошибка получения файла из формы",
				"error", err,
				"remote_addr", r.RemoteAddr,
				"method", r.Method,
				"path", r.URL.Path,
			)
			http.Error(w, "Не удалось получить файл", http.StatusBadRequest)
			return
		}
		defer file.Close()
		videoName := handler.Filename

		// Логируем информацию о файле
		slog.Info("Получен файл для загрузки",
			"filename", videoName,
			"size", handler.Size,
			"content_type", handler.Header.Get("Content-Type"),
			"remote_addr", r.RemoteAddr,
		)

		// Валидация расширения
		allowedExtensions := map[string]bool{
			".mp4": true, ".avi": true, ".mov": true,
			".mkv": true, ".webm": true,
		}
		ext := filepath.Ext(videoName)
		if !allowedExtensions[ext] {
			slog.Warn("Запрещённое расширение файла",
				"extension", ext,
				"filename", videoName,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, fmt.Sprintf("Тип файла %s не разрешён", ext), http.StatusBadRequest)
			return
		}

		// Генерируем безопасное уникальное имя
		uniqueName := rand.Text()
		if err != nil {
			slog.Error("Не удалось сгенерировать имя файла",
				"error", err,
				"original_filename", videoName,
			)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}
		filename := uniqueName + ext
		filePath := filepath.Join(config.TemporaryDir, filename)
		dst, err := os.Create(filePath)
		if err != nil {
			slog.Error("Ошибка создания файла на сервере",
				"error", err,
				"filepath", filePath,
			)
			http.Error(w, "Не удалось сохранить файл", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Копируем содержимое
		_, err = io.Copy(dst, file)
		if err != nil {
			slog.Error("Ошибка записи файла на диск",
				"error", err,
				"filename", filename,
				"original_filename", videoName,
			)
			http.Error(w, "Ошибка записи файла", http.StatusInternalServerError)
			return
		}

		// Успешный ответ
		slog.Info("Видео успешно загружено",
			"original_filename", videoName,
			"stored_filename", filename,
			"size", handler.Size,
		)

		videoStorage.InsertVideo(videoName, uniqueName)
		go func(filename string) {
			slog.Info("Запускается фоновая конвертация в HLS", "filename", filename)
			hlsErr := utils.GenerateAdaptiveHLS(config.TemporaryDir, config.UploadDir, filename)
			if hlsErr != nil {
				slog.Error("Ошибка конвертации MP4 в HLS",
					"error", hlsErr,
					"mp4_filename", filename,
				)
				videoStorage.DeleteVideoByFileName(uniqueName)

			} else {
				slog.Info("HLS конвертация завершена успешно", "mp4_filename", filename)
			}
			os.Remove(filePath)

		}(filename)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message":           "Видео успешно загружено и начинается обработка.",
			"filename_for_hls":  strings.TrimSuffix(filename, filepath.Ext(filename)),
			"original_filename": videoName,
		})

	}
}

func startConvertVideo(filePath, supportedFilePath string, done chan error) {
	err := utils.ConvertToMP4(filePath, supportedFilePath)
	done <- err
	os.Remove(filePath)
	if err != nil {
		os.Remove(supportedFilePath)
	}
}
