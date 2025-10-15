package video

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"video/config"
	"video/database"

	"log/slog"
)

// Delete удаляет видео по имени файла.
// Ожидает GET-параметр: ?file_name=имя_файла.mp4
func Delete(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		videoName := r.URL.Query().Get("file_name")
		if videoName == "" {
			slog.Warn("Параметр file_name не указан",
				"remote_addr", r.RemoteAddr,
				"method", r.Method,
				"path", r.URL.Path,
			)
			http.Error(w, "Missing required parameter: file_name", http.StatusBadRequest)
			return
		}

		// Ищем видео в БД
		video, err := db.GetVideoByFileName(videoName)
		if err != nil {
			slog.Error("Видео не найдено в базе данных",
				"video_name", videoName,
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Video not found in database", http.StatusNotFound)
			return
		}
		filePath := filepath.Join(config.UploadDir, video.FileName)
		// Проверяем наличие файла на диске
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			slog.Warn("Файл видео отсутствует на диске — удаляем только из БД",
				"video_name", videoName,
				"file_path", video.FileName,
				"remote_addr", r.RemoteAddr,
			)

			// Удаляем из БД, даже если файла нет
			if err := db.DeleteVideoByID(video.ID); err != nil {
				slog.Error("Не удалось удалить видео из БД (файл уже отсутствует)",
					"video_id", video.ID,
					"error", err,
					"remote_addr", r.RemoteAddr,
				)
				http.Error(w, "Failed to clean up database record", http.StatusInternalServerError)
				return
			}

			// Ответ: файл уже отсутствовал, но запись в БД удалена
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{
				"message":  "Video record cleaned up (file was already missing)",
				"filename": video.FileName,
			})
			return
		}

		// Удаляем файл с диска
		if err := os.RemoveAll(filePath); err != nil {
			slog.Error("Не удалось удалить файл с диска",
				"file_path", video.FileName,
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Failed to delete video file", http.StatusInternalServerError)
			return
		}

		// Удаляем запись из БД
		if err := db.DeleteVideoByID(video.ID); err != nil {
			slog.Error("Не удалось удалить запись из БД после удаления файла",
				"video_id", video.ID,
				"file_path", video.FileName,
				"error", err,
				"remote_addr", r.RemoteAddr,
			)
			// ⚠️ Файл удалён, но БД не обновлена — это опасное состояние!
			// Можно вернуть 500 и залогировать тревогу
			http.Error(w, "File deleted but database cleanup failed — manual intervention required", http.StatusInternalServerError)
			return
		}

		// Успешный ответ
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"message":  "Video successfully deleted",
			"filename": video.FileName,
		}); err != nil {
			slog.Error("Не удалось отправить JSON-ответ", "error", err)
		}
	}
}
