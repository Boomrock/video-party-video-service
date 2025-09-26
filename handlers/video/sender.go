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

// get?
// file_name - имя файла видео String
func Sender(streamer streamer.Streamer, database *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Обрабатываем запрос
		// Делегируем получение куска видео
		// Отправляем кусок видео

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
			http.Error(w, "video not found", http.StatusBadRequest)
			return
		}

		rangeHeader := r.Header.Get("Range")
		var start, end int

		if rangeHeader == "" {
			// По умолчанию — первые 1 МБ
			start = 0
			end = 1024*1024 - 1
		} else {
			// Разбираем заголовок Range: "bytes=start-end"
			rangeParts := strings.TrimPrefix(rangeHeader, "bytes=")
			rangeValues := strings.Split(rangeParts, "-")
			var err error

			// Начальный байт
			start, err = strconv.Atoi(rangeValues[0])
			if err != nil {
				slog.Error("Некорректный начальный байт в заголовке Range",
					"диапазон", rangeHeader,
					"ошибка", err,
					"удалённый_адрес", r.RemoteAddr,
				)
				http.Error(w, "Invalid start byte", http.StatusBadRequest)
				return
			}

			// Конечный байт или по умолчанию
			if len(rangeValues) > 1 && rangeValues[1] != "" {
				end, err = strconv.Atoi(rangeValues[1])
				if err != nil {
					slog.Error("Некорректный конечный байт в заголовке Range",
						"диапазон", rangeHeader,
						"ошибка", err,
						"удалённый_адрес", r.RemoteAddr,
					)
					http.Error(w, "Invalid end byte", http.StatusBadRequest)
					return
				}
			} else {
				end = start + 1024*1024 - 1 // По умолчанию — 1 МБ
			}
		}

		// Убедимся, что конец не выходит за размер видео
		if end >= video.Size {
			end = video.Size - 1
		}

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

		// Устанавливаем заголовки и отправляем фрагмент
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, video.Size))
		w.Header().Set("Accept-Ranges", "bytes")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(videoData)))
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
			// Нельзя отправить ошибку — ответ уже начат
		}
	}
}
