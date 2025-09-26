package video

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"video/database"
	"video/streamer"

	"log/slog"
)

// GET /video?file_name=...
func Sender(streamer streamer.Streamer, database *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		videoName := r.URL.Query().Get("file_name")
		if videoName == "" {
			slog.Error("Отсутствует обязательный параметр: file_name",
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			http.Error(w, "Missing required parameter: file_name", http.StatusBadRequest)
			return
		}

		video, err := database.GetVideoByFileName(videoName)
		if err != nil {
			slog.Error("Видео не найдено в базе данных",
				"имя_видео", videoName,
				"ошибка", err,
				"удалённый_адрес", r.RemoteAddr,
			)
			http.Error(w, "video not found", http.StatusNotFound) // ✅ 404 вместо 400
			return
		}

		rangeHeader := r.Header.Get("Range")
		var start, end int64

		if rangeHeader == "" {
			// По умолчанию — первые 1 МБ
			start = 0
			end = 1024*1024 - 1
		} else {
			rangeParts := strings.TrimPrefix(rangeHeader, "bytes=")
			parts := strings.Split(rangeParts, "-")
			if len(parts) != 2 {
				slog.Error("Некорректный формат заголовка Range",
					"диапазон", rangeHeader,
					"удалённый_адрес", r.RemoteAddr,
				)
				http.Error(w, "Invalid range format", http.StatusBadRequest)
				return
			}

			// Парсим start
			if parts[0] == "" {
				slog.Error("Отсутствует начальный байт в Range",
					"диапазон", rangeHeader,
					"удалённый_адрес", r.RemoteAddr,
				)
				http.Error(w, "Invalid range: missing start", http.StatusBadRequest)
				return
			}
			start, err = strconv.ParseInt(parts[0], 10, 64)
			if err != nil || start < 0 {
				slog.Error("Некорректный начальный байт в заголовке Range",
					"диапазон", rangeHeader,
					"ошибка", err,
					"удалённый_адрес", r.RemoteAddr,
				)
				http.Error(w, "Invalid start byte", http.StatusBadRequest)
				return
			}

			// Парсим end
			if parts[1] == "" {
				// Если конец не указан — до конца файла
				end = video.Size - 1
			} else {
				end, err = strconv.ParseInt(parts[1], 10, 64)
				if err != nil || end < start {
					slog.Error("Некорректный конечный байт в заголовке Range",
						"диапазон", rangeHeader,
						"ошибка", err,
						"удалённый_адрес", r.RemoteAddr,
					)
					http.Error(w, "Invalid end byte", http.StatusBadRequest)
					return
				}
			}
		}

		// 🔒 Проверка: start за пределами файла → 416
		if start >= video.Size {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", video.Size))
			slog.Warn("Запрошенный диапазон вне размера видео",
				"имя_видео", videoName,
				"start", start,
				"size", video.Size,
				"удалённый_адрес", r.RemoteAddr,
			)
			http.Error(w, "Requested range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
			return
		}

		// 🔒 Обрезаем end по размеру файла
		if end >= video.Size {
			end = video.Size - 1
		}

		// Теперь безопасно читаем
		videoData, err := streamer.Seek(video, start, end)
		if err != nil {
			slog.Error("Ошибка при получении фрагмента видео",
				"имя_видео", videoName,
				"начало", start,
				"конец", end,
				"ошибка", err,
			)
			http.Error(w, fmt.Sprintf("Error retrieving video: %v", err), http.StatusInternalServerError)
			return
		}

		// ✅ Устанавливаем правильные заголовки
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, video.Size))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", strconv.Itoa(len(videoData)))
		w.Header().Set("Content-Type", "video/mp4")
		w.WriteHeader(http.StatusPartialContent)

		_, err = w.Write(videoData)
		if err != nil {
			slog.Error("Ошибка при потоковой передаче видео клиенту",
				"имя_видео", videoName,
				"начало", start,
				"конец", end,
				"размер", len(videoData),
				"ошибка", err,
				"удалённый_адрес", r.RemoteAddr,
			)
			// Нельзя изменить статус — ответ уже начат
		}
	}
}