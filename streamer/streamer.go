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
	Seek(video *database.Video, start, end int) ([]byte, error)
}

// Мок для хранения видео
type FileStreamer struct {
	i int
}

func (f *FileStreamer) Seek(video *database.Video, start, end int) ([]byte, error) {
	// 1. Открываем файл
	file, err := os.Open(path.Join(config.UploadDir, video.FileName))
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %w", err)
	}
	defer file.Close() // Важно закрыть файл
	// 2. Проверяем корректность позиций
	if start < 0 || end < 0 {
		return nil, fmt.Errorf("некорректные позиции: start=%d, end=%d (должны быть неотрицательными)", start, end)
	}
	if start > end {
		return nil, fmt.Errorf("некорректные позиции: start=%d, end=%d (start не может быть больше end)", start, end)
	}

	// 3. Получаем размер файла для проверки границ
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить информацию о файле: %w", err)
	}
	fileSize := fileInfo.Size()

	if int64(start) >= fileSize {
		return nil, fmt.Errorf("начальная позиция (%d) находится за пределами файла (размер: %d байт)", start, fileSize)
	}
	// end может быть равен fileSize - 1 (последний байт) или меньше
	if int64(end) >= fileSize {
		end = int(fileSize)
	}

	// 4. Вычисляем количество байт для чтения
	// end - start + 1, потому что позиции включают и start, и end
	count := int64(end - start + 1)

	// 5. Создаем буфер для хранения данных
	buf := make([]byte, count)

	// 6. Читаем данные из файла, начиная с позиции start
	// ReadAt читает ровно len(buf) байт, начиная с указанной позиции
	_, err = file.ReadAt(buf, int64(start))
	if err != nil {
		// В теории, при правильной проверке границ, io.EOF не должна возникнуть,
		// но всё же обработаем её на всякий случай.
		if err == io.EOF {
			// Это может произойти, если файл был усечен между Stat() и ReadAt()
			return nil, fmt.Errorf("неожиданный конец файла при чтении: %w", err)
		}
		return nil, fmt.Errorf("ошибка чтения файла: %w", err)
	}
	return buf, nil
}
