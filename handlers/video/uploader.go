package video

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"video/config"
	database "video/database"

	"log/slog" // <-- добавлен
)
//Метод загрузки видео на сервер 
//post?video
func Upload(db *database.DB) http.HandlerFunc {
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

		// Логируем информацию о файле
		slog.Info("Получен файл для загрузки",
			"filename", handler.Filename,
			"size", handler.Size,
			"content_type", handler.Header.Get("Content-Type"),
			"remote_addr", r.RemoteAddr,
		)

		// Валидация расширения
		allowedExtensions := map[string]bool{
			".mp4": true, ".avi": true, ".mov": true,
			".mkv": true, ".webm": true,
		}
		ext := filepath.Ext(handler.Filename)
		if !allowedExtensions[ext] {
			slog.Warn("Запрещённое расширение файла",
				"extension", ext,
				"filename", handler.Filename,
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
				"original_filename", handler.Filename,
			)
			http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
			return
		}
		filename := uniqueName + ext

		// Создаём путь для сохранения
		filePath := filepath.Join(config.UploadDir, filename)
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
				"original_filename", handler.Filename,
			)
			http.Error(w, "Ошибка записи файла", http.StatusInternalServerError)
			return
		}

		// Сохраняем метаданные в БД
		err = db.InsertVideo(handler.Filename, filename, int(handler.Size))
		if err != nil {
			slog.Error("Ошибка сохранения видео в базу данных",
				"error", err,
				"stored_filename", filename,
				"original_filename", handler.Filename,
				"size", handler.Size,
			)
			// Опционально: удалить файл, если не удалось записать в БД
			os.Remove(filePath) // Чистим мусор
			http.Error(w, "Ошибка сохранения данных", http.StatusInternalServerError)
			return
		}

		// Успешный ответ
		slog.Info("Видео успешно загружено",
			"original_filename", handler.Filename,
			"stored_filename", filename,
			"size", handler.Size,
		)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message":  "Видео успешно загружено",
			"filename": filename,
		})
	}
}
