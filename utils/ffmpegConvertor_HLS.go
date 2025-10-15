package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Структура для определения настроек качества
type HLSQuality struct {
	Resolution   string // Например, "854x480"
	VideoBitrate string // Например, "1M" (1 Mbps)
	AudioBitrate string // Например, "96k" (96 kbps)
	BaseName     string // Например, "480p"
	Bandwidth    int    // Суммарный битрейт (видео + аудио) для #EXT-X-STREAM-INF, в bps
}

func generateSingleQualityHLS(
	inputPath string,
	outputPathDir string,
	segmentDuration int,
	playlistType string,
	resolution string,
	videoBitrate string,
	audioBitrate string,
	outputBaseName string,
) error {
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		err := fmt.Errorf("входной файл не найден: %s", inputPath)
		return err
	}
	if err := os.MkdirAll(outputPathDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать выходную директорию: %w", err)
	}

	outputPlaylistPath := filepath.Join(outputPathDir, fmt.Sprintf("%s.m3u8", outputBaseName))

	args := []string{
		"-i", inputPath,
		"-c:v", "libx264",
		"-b:v", videoBitrate,
		"-preset", "ultrafast",
		"-vf", fmt.Sprintf("scale=%s", resolution),
		"-c:a", "aac",
		"-b:a", audioBitrate,
		"-f", "hls",
		"-hls_list_size", "0",
		"-hls_flags", "temp_file",
		"-hls_time", strconv.Itoa(segmentDuration),
		"-hls_playlist_type", playlistType,
		"-hls_segment_filename", filepath.Join(outputPathDir, fmt.Sprintf("%s_%%03d.ts", outputBaseName)),
		outputPlaylistPath,
	}


	cmd := exec.Command("ffmpeg", args...)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ошибка при выполнении ffmpeg для HLS качества %s: %w", outputBaseName, err)
	}

	return nil
}

// GenerateAdaptiveHLS генерирует HLS-потоки для нескольких качеств и мастер-плейлист.
// inputFolder - папка где лежит оригинальный файл
// outputFoler - папка где будет храниться сгенерированный HLS плейлист
// originalFileName - имя оригинального файла (например, "my_awesome_video.mp4").
// HLS файлы будут сгенерированы в поддиректорию с именем, соответствующим originalFileName без расширения.
func GenerateAdaptiveHLS(inputFolder, outputFoler, originalFileName string) error {
	// Имя папки для HLS-файлов будет именем файла без расширения
	videoFolderName := strings.TrimSuffix(originalFileName, filepath.Ext(originalFileName))
	
	// Полный путь к оригинальному  файлу
	inputPath := filepath.Join(inputFolder, originalFileName)
	
	// Общая директория для всех HLS файлов этого видео
	outputPathDir := filepath.Join(outputFoler, videoFolderName)

	// 1. Проверки перед началом
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		err := fmt.Errorf("входной файл не найден: %s", inputPath)
		return err
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg не установлен или не в PATH: %v", err)
	}
	if err := os.MkdirAll(outputPathDir, 0755); err != nil {
		return fmt.Errorf("не удалось создать выходную директорию %s: %w", outputPathDir, err)
	}

	// 2. Определяем конфигурации для разных качеств (можно вынести в config)
	qualities := []HLSQuality{
		{Resolution: "854x480", VideoBitrate: "1M", AudioBitrate: "96k", BaseName: "480p", Bandwidth: 1100000},
		//{Resolution: "1280x720", VideoBitrate: "2M", AudioBitrate: "128k", BaseName: "720p", Bandwidth: 2150000},
		//{Resolution: "1920x1080", VideoBitrate: "4M", AudioBitrate: "192k", BaseName: "1080p", Bandwidth: 4250000},
	}

	segmentDuration := 10 // Длительность каждого сегмента в секундах
	playlistType := "event" // Тип плейлиста: "vod" (Video On Demand)

	var generatedPlaylists []HLSQuality
	err := func() error{
		// 3. Генерируем HLS-поток для каждого качества
		for _, q := range qualities {
			err := generateSingleQualityHLS(
				inputPath,
				outputPathDir,
				segmentDuration,
				playlistType,
				q.Resolution,
				q.VideoBitrate,
				q.AudioBitrate,
				q.BaseName,
			)
			if err == nil {
				generatedPlaylists = append(generatedPlaylists, q)

			}
		}
	

	
		// 4. Создаем мастер-плейлист
		masterPlaylistPath := filepath.Join(outputPathDir, "main.m3u8")
		masterPlaylistFile, err := os.Create(masterPlaylistPath)
		if err != nil {
			return fmt.Errorf("не удалось создать мастер-плейлист %s: %w", masterPlaylistPath, err)
		}
		defer masterPlaylistFile.Close()

		_, err = fmt.Fprintf(masterPlaylistFile, "#EXTM3U\n")
		if err != nil {
			return fmt.Errorf("ошибка записи в мастер-плейлист: %w", err)
		}
		_, err = fmt.Fprintf(masterPlaylistFile, "#EXT-X-VERSION:3\n") 
		if err != nil {
			return fmt.Errorf("ошибка записи в мастер-плейлист: %w", err)
		}

		for _, q := range generatedPlaylists {
			// Извлекаем чистый числовой битрейт для Bandwidth из VideoBitrate и AudioBitrate
			videoBitrateVal := parseBitrateToBPS(q.VideoBitrate)
			audioBitrateVal := parseBitrateToBPS(q.AudioBitrate)

			// Суммарный битрейт видео и аудио для параметра BANDWIDTH
			bandwidth := videoBitrateVal + audioBitrateVal

			
			_, err := fmt.Fprintf(masterPlaylistFile, 
				"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%s,CODECS=\"avc1.42e01e,mp4a.40.2\"\n", 
				bandwidth, 
				q.Resolution)
			if err != nil {
				return fmt.Errorf("ошибка записи в мастер-плейлист для качества %s: %w", q.BaseName, err)
			}

			_, err = fmt.Fprintf(masterPlaylistFile, "%s.m3u8\n", q.BaseName)
			if err != nil {
				return fmt.Errorf("ошибка записи в мастер-плейлист для качества %s: %w", q.BaseName, err)
			}
		}
		return nil
	}()
	if err != nil {
		removeErr := os.RemoveAll(outputPathDir)
		if removeErr != nil {
			return fmt.Errorf("ошибка обработки видео: %v; дополнительно: не удалось удалить директорию: %v", err, removeErr)
		}
		return fmt.Errorf("ошибка обработки видео: %w", err)
	}
	return nil
}

// parseBitrateToBPS конвертирует строку битрейта (например, "1M", "96k") в биты в секунду.
func parseBitrateToBPS(bitrate string) int {
	if len(bitrate) < 2 {
		return 0
	}
	valStr := bitrate[:len(bitrate)-1]
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0
	}

	switch strings.ToLower(bitrate[len(bitrate)-1:]) {
	case "m":
		return int(val * 1000000)
	case "k":
		return int(val * 1000)
	default:
		return int(val) // Если нет суффикса, считаем, что уже в bps
	}
}