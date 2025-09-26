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
	"time"
	"video/config"
	database "video/database"
	"video/utils"

	"log/slog" // <-- добавлен
)

func getFilenameWithoutExt(filePath string) string {
	// 1. Получаем только имя файла (без пути)
	filename := filepath.Base(filePath)

	// 2. Удаляем расширение
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	return nameWithoutExt
}

const supExt string = ".mp4"

// Метод загрузки видео на сервер
// post?video
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
		supportedFileName := uniqueName + supExt
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
				"original_filename", videoName,
			)
			http.Error(w, "Ошибка записи файла", http.StatusInternalServerError)
			return
		}

		// Сохраняем метаданные в БД
		err = db.InsertVideo(videoName, supportedFileName, handler.Size) // Мы сохраням другое расширение, так как нужно еще сконвертить это в поддерживамое расширение
		if err != nil {
			slog.Error("Ошибка сохранения видео в базу данных",
				"error", err,
				"stored_filename", filename,
				"original_filename", videoName,
				"size", handler.Size,
			)
			// Опционально: удалить файл, если не удалось записать в БД
			os.Remove(filePath) // Чистим мусор
			http.Error(w, "Ошибка сохранения данных", http.StatusInternalServerError)
			return
		}

		// Успешный ответ
		slog.Info("Видео успешно загружено",
			"original_filename", videoName,
			"stored_filename", filename,
			"size", handler.Size,
		)
		if ext != supExt {
			supportedFilePath := filepath.Join(config.UploadDir, supportedFileName)
			done := make(chan error, 1)
			tick := time.NewTicker(100 * time.Millisecond)
			go startConvertVideo(filePath, supportedFilePath, done)
			var info os.FileInfo
			for { // подождем создания файла а потом говорим что все готово
				select {
				case err := <-done:
					if err != nil {
						slog.Error("Ошибка сохранения видео",
							"error", err,
							"stored_filename", filename,
							"original_filename", videoName,
						)
						http.Error(w, "Ошибка сохранения данных", http.StatusInternalServerError)
					}
					return
				case <-tick.C:
					//сморим раз 0.1 секунды как там у нас файл
				}
				info, err = os.Stat(supportedFilePath) // если файл создан говорим что его можно читать
				if err == nil {
					break
				}
			}

			err = db.UpdateVideoSize(supportedFileName, info.Size())
			if err != nil {
				slog.Error("Ошибка сохранения видео",
					"error", err,
					"stored_filename", filename,
					"original_filename", videoName,
					"size", info.Size(),
				)
				http.Error(w, "Ошибка сохранения данных", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"message":  "Видео успешно загружено",
			"filename": filename,
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
