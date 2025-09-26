package streamer

import (
	"fmt"
	"io"
	"os"
	"path"
	"video/config"
	"video/database"
)

type Streamer interface {
	Seek(video *database.Video, start, end int64) ([]byte, error)
}

type FileStreamer struct{}

// Seek возвращает фрагмент файла [start, end] (включительно)
func (f *FileStreamer) Seek(video *database.Video, start, end int64) ([]byte, error) {
	if start < 0 || end < 0 {
		return nil, fmt.Errorf("некорректные позиции: start=%d, end=%d (должны быть неотрицательными)", start, end)
	}
	if start > end {
		return nil, fmt.Errorf("некорректные позиции: start=%d, end=%d (start не может быть больше end)", start, end)
	}

	fullPath := path.Join(config.UploadDir, video.FileName)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл %q: %w", fullPath, err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить информацию о файле %q: %w", fullPath, err)
	}
	fileSize := fileInfo.Size()

	// Если start за пределами файла — ошибка (416 Range Not Satisfiable)
	if start >= fileSize {
		return nil, fmt.Errorf("начальная позиция (%d) за пределами файла (размер: %d)", start, fileSize)
	}

	// Обрезаем end по размеру файла
	if end >= fileSize {
		end = fileSize - 1
	}

	// Вычисляем количество байт (осторожно с переполнением)
	count := end - start + 1
	if count <= 0 {
		return []byte{}, nil
	}

	// Защита от слишком больших запросов (опционально)
	const maxChunkSize = 100 * 1024 * 1024 // 100 MB
	if count > maxChunkSize {
		return nil, fmt.Errorf("запрашиваемый фрагмент слишком большой: %d байт (максимум %d)", count, maxChunkSize)
	}

	buf := make([]byte, count)
	n, err := file.ReadAt(buf, start)
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("ошибка чтения файла %q: %w", fullPath, err)
	}
	if n != len(buf) {
		// Файл был изменён во время чтения — возвращаем то, что прочитали
		return buf[:n], nil
	}

	return buf, nil
}