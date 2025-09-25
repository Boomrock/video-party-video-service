package video

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"video/database"
)
//get?
func GetAllVideo(database *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		videos, err := database.GetAllVideos()
		if err != nil {
			slog.Error(fmt.Sprintf("Не удалось получить видео"),
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
				"ошибка", err.Error(),
			)
			http.Error(w, "Videos not get", http.StatusNotFound)
			return
		}

		// Устанавливаем тип содержимого
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		err = json.NewEncoder(w).Encode(videos)
		if err != nil {
			slog.Error("Ошибка при отправке ответа с ключом комнаты",
				"error", err,
				"удалённый_адрес", r.RemoteAddr,
				"метод", r.Method,
				"путь", r.URL.Path,
			)
			// Нельзя вызвать http.Error после начала записи в w
			return
		}
	}
}
