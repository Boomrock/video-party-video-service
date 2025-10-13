package database

import (
	"fmt"
	"strings"
)

// Video представляет структуру данных видео.
type Video struct {
	ID        int    // ID обычно не превышает int
	VideoName string // имя видео при внесении в систему
	FileName  string // имя файла в системе
	Size      int64  // ✅ Размер в байтах — может быть >2 ГБ
	HLSConverted bool // признак, что видео уже преобразовано в HLS
	HLSErrorMessage string
}

// CreateVideosTable создает таблицу 'videos', если она еще не существует.
func (db *DB) CreateVideosTable() error {
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS videos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		video_name TEXT NOT NULL,
		file_name TEXT UNIQUE, 
		size INTEGER NOT NULL,  -- SQLite: INTEGER = 64-bit signed
		hls_converted BOOLEAN DEFAULT FALSE,
		hls_error_message TEXT
	);`
	_, err := db.conn.Exec(createTablesSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблицы 'videos': %w", err)
	}
	fmt.Println("Таблица 'videos' готова.")
	return nil
}

// InsertVideo добавляет новую запись видео в таблицу 'videos'.
func (db *DB) InsertVideo(videoName, fileName string, size int64) error {
	insertSQL := `INSERT INTO videos (video_name, file_name, size) VALUES (?, ?, ?)`
	_, err := db.conn.Exec(insertSQL, videoName, fileName, size)
	if err != nil {
		return fmt.Errorf("ошибка вставки видео (video_name: '%s', file_name: '%s'): %w", videoName, fileName, err)
	}
	fmt.Printf("Видео добавлено: '%s' -> '%s' (размер: %d байт)\n", videoName, fileName, size)
	return nil
}

// UpdateVideoSize обновляет размер видео по имени файла.
func (db *DB) UpdateVideoSize(fileName string, size int64) error {
	updateSQL := `UPDATE videos SET size = ? WHERE file_name = ?`
	result, err := db.conn.Exec(updateSQL, size, fileName)
	if err != nil {
		return fmt.Errorf("ошибка обновления размера видео (file_name: '%s', size: %d): %w", fileName, size, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества затронутых строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("видео с file_name '%s' не найдено для обновления размера", fileName)
	}

	return nil
}

// GetAllVideos получает все записи из таблицы 'videos'.
func (db *DB) GetAllVideos() ([]Video, error) {
	querySQL := `SELECT id, video_name, file_name, size FROM videos`
	rows, err := db.conn.Query(querySQL)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer rows.Close()

	var videos []Video
	for rows.Next() {
		var v Video
		err := rows.Scan(&v.ID, &v.VideoName, &v.FileName, &v.Size)
		if err != nil {
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		videos = append(videos, v)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по строкам: %w", err)
	}

	return videos, nil
}

// GetVideoByID получает видео по его ID.
func (db *DB) GetVideoByID(id int) (*Video, error) {
	querySQL := `SELECT id, video_name, file_name, size FROM videos WHERE id = ?`
	row := db.conn.QueryRow(querySQL, id)

	var v Video
	err := row.Scan(&v.ID, &v.VideoName, &v.FileName, &v.Size)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения видео по ID %d: %w", id, err)
	}

	return &v, nil
}

// GetVideoByFileName получает видео по его file_name.
func (db *DB) GetVideoByFileName(fileName string) (*Video, error) {
	querySQL := `SELECT id, video_name, file_name, size FROM videos WHERE file_name = ?`
	row := db.conn.QueryRow(querySQL, fileName)

	var v Video
	err := row.Scan(&v.ID, &v.VideoName, &v.FileName, &v.Size)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения видео по file_name '%s': %w", fileName, err)
	}

	return &v, nil
}

// UpdateVideo обновляет video_name, file_name и/или size видео по ID.
func (db *DB) UpdateVideo(id int, newVideoName, newFileName string, size int64) error {
	var updates []string
	var args []interface{}

	if newVideoName != "" {
		updates = append(updates, "video_name = ?")
		args = append(args, newVideoName)
	}
	if newFileName != "" {
		updates = append(updates, "file_name = ?")
		args = append(args, newFileName)
	}

	// Всегда обновляем size, если передан (можно сделать условно)
	updates = append(updates, "size = ?")
	args = append(args, size)

	if len(updates) == 1 {
		fmt.Printf("Нет данных для обновления видео с ID %d.\n", id)
		return nil
	}

	args = append(args, id)
	querySQL := fmt.Sprintf("UPDATE videos SET %s WHERE id = ?", strings.Join(updates, ", "))
	result, err := db.conn.Exec(querySQL, args...)
	if err != nil {
		return fmt.Errorf("ошибка обновления видео с ID %d: %w", id, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества затронутых строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("видео с ID %d не найдено для обновления", id)
	}

	fmt.Printf("Видео с ID %d обновлено.\n", id)
	return nil
}

// DeleteVideoByID удаляет видео по его ID.
func (db *DB) DeleteVideoByID(id int) error {
	deleteSQL := `DELETE FROM videos WHERE id = ?`
	result, err := db.conn.Exec(deleteSQL, id)
	if err != nil {
		return fmt.Errorf("ошибка удаления видео: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества затронутых строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("видео с ID %d не найдено для удаления", id)
	}

	fmt.Printf("Видео с ID %d удалено.\n", id)
	return nil
}

// DeleteVideoByFileName удаляет видео по его file_name.
func (db *DB) DeleteVideoByFileName(fileName string) error {
	deleteSQL := `DELETE FROM videos WHERE file_name = ?`
	result, err := db.conn.Exec(deleteSQL, fileName)
	if err != nil {
		return fmt.Errorf("ошибка удаления видео по file_name: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("ошибка получения количества затронутых строк: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("видео с file_name '%s' не найдено для удаления", fileName)
	}

	fmt.Printf("Видео с file_name '%s' удалено.\n", fileName)
	return nil
}


func (db *DB) UpdateHLSConversionStatus(mp4FileName string, converted bool, errorMessage string) error {
	_, err := db.conn.Exec("UPDATE videos SET hls_converted = ?, hls_error_message = ? WHERE file_name = ?", converted, errorMessage, mp4FileName)
    return err
}