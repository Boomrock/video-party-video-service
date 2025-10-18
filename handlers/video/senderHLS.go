package video

import (
	// "fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"video/config"

	"github.com/go-chi/chi/v5"
)

// HLSHandler обслуживает HLS-файлы (манифесты .m3u8 и сегменты .ts).
// Ожидаемый формат URL: /hls/{video_id}/{filename.m3u8_or_ts}
func HLSHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Отрезаем префикс "/hls/"
		relativePath := chi.URLParam(r, "*")

		if relativePath == "" { // Если префикс не был найден, значит путь неверный
			slog.Error("путь не может быть пустым",
				"удалённый_адрес", r.RemoteAddr,
				"полный_путь_URL", r.URL.Path,
			)
			http.Error(w, "Invalid HLS path", http.StatusBadRequest)
			return
		}

		// Можно извлечь video_id, если нужно для логирования или дополнительной проверки
		parts := strings.Split(relativePath, "/")
		if len(parts) < 2 {
			slog.Error("Неверный HLS-путь, отсутствует video_id или имя файла",
				"удалённый_адрес", r.RemoteAddr,
				"относительный_путь", relativePath,
			)
			http.Error(w, "Invalid HLS path", http.StatusBadRequest)
			return
		}
		// videoID := parts[0] // Первый элемент - это video_id
		fileName := strings.Join(parts[1:], "/") // Остальное - это имя файла и его подпуть

		// fmt.Println("Debug: videoID:", videoID, "fileName:", fileName)

		// filepath.Join автоматически обработает config.UploadDir/videoID/fileName
		filePath := filepath.Join(config.UploadDir, relativePath)
		// fmt.Println("Debug: filePath:", filePath)

		fileInfo, err := os.Stat(filePath)
		if os.IsNotExist(err) {
			slog.Warn("HLS файл не найден",
				"полный_путь_файла", filePath,
				"удалённый_адрес", r.RemoteAddr,
			)
			http.Error(w, "HLS file not found", http.StatusNotFound)
			return
		}
		if err != nil {
			slog.Error("Ошибка при доступе к HLS файлу",
				"полный_путь_файла", filePath,
				"ошибка", err,
				"удалённый_адрес", r.RemoteAddr,
			)
			http.Error(w, "Error accessing HLS file", http.StatusInternalServerError)
			return
		}
		if fileInfo.IsDir() {
			slog.Warn("Запрошенный HLS путь является директорией, а не файлом",
				"полный_путь_файла", filePath,
				"удалённый_адрес", r.RemoteAddr,
			)
			http.Error(w, "Not a file", http.StatusForbidden)
			return
		}

		// Устанавливаем правильный Content-Type
		if strings.HasSuffix(fileName, ".m3u8") { // Проверяем fileName, т.к. hlsRelativePath содержит videoID
			w.Header().Set("Content-Type", "application/x-mpegURL")
		} else if strings.HasSuffix(fileName, ".fmp4") {
			w.Header().Set("Content-Type", "video/iso.segment")
		} else {
			slog.Warn("Неизвестное расширение файла HLS, Content-Type не установлен явно",
				"путь_файла", relativePath, // Здесь лучше использовать relativePath
				"удалённый_адрес", r.RemoteAddr,
			)
		}

		http.ServeFile(w, r, filePath)

		slog.Info("HLS файл успешно отдан",
			"файл", relativePath,
			"удалённый_адрес", r.RemoteAddr,
		)
	}
}
