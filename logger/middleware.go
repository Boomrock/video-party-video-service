package logger

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func Middlerware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Обёртка для отслеживания статуса и объёма ответа
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Выполняем следующий обработчик
		next.ServeHTTP(ww, r)

		// Собираем данные для лога
		duration := time.Since(start)
		status := ww.Status()
		size := ww.BytesWritten()

		// Определяем уровень лога в зависимости от статуса
		var level slog.Level
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		default:
			level = slog.LevelInfo
		}

		// Формируем сообщение
		msg := "Обработан HTTP-запрос"
		if status >= 500 {
			msg = "Ошибка сервера при обработке запроса"
		} else if status >= 400 {
			msg = "Ошибка клиента при запросе"
		}

		// Логируем
		slog.Log(r.Context(), level, msg,
			"метод", r.Method,
			"путь", r.URL.Path,
			"заголовок_range", r.Header.Get("Range"), // полезно для стриминга
			"статус", status,
			"размер_ответа", size,
			"время_обработки_ms", duration.Milliseconds(),
			"удалённый_адрес", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}
