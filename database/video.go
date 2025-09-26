package database

import (
	"fmt"
	"strings"
)

// Video представляет структуру данных видео.
type Video struct {
	ID        int
	VideoName string // имя видео при внесении в систему
	FileName  string // имя файла в системе
	Size      int
}

// CreateVideosTable создает таблицу 'videos', если она еще не существует.
// Таблица содержит поля:
// - id (INTEGER PRIMARY KEY AUTOINCREMENT)
// - video_name (TEXT NOT NULL)
// - file_name (TEXT UNIQUE)
func (db *DB) CreateVideosTable() error {
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS videos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		video_name TEXT NOT NULL,
		file_name TEXT UNIQUE, 
		size INTEGER NOT NULL
	);`
	_, err := db.conn.Exec(createTablesSQL)
	if err != nil {
		return fmt.Errorf("ошибка создания таблиц: %w", err)
	}
	fmt.Println("Таблица 'videos' готова.")
	return nil
}

// InsertVideo добавляет новую запись видео в таблицу 'videos'.
// fileName должен быть уникальным.
func (db *DB) InsertVideo(videoName, fileName string, size int) error {
	insertSQL := `INSERT INTO videos (video_name, file_name, size) VALUES (?, ?, ?)`
	_, err := db.conn.Exec(insertSQL, videoName, fileName, size)
	if err != nil {
		return fmt.Errorf("ошибка вставки видео (video_name: '%s', file_name: '%s'): %w", videoName, fileName, err)
	}
	fmt.Printf("Видео добавлено: '%s' -> '%s'\n", videoName, fileName)
	return nil
}
func (db *DB) UpdateVideoSize(fileName string, size int) error {
	updateSQL := `UPDATE videos SET size = ? WHERE file_name = ?`

	_, err := db.conn.Exec(updateSQL, size, fileName)
	if err != nil {
		return fmt.Errorf("ошибка вставки обновления размера видео (video_name: '%s', file_name: '%d'): %w", fileName, size, err)
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
	defer rows.Close() // Важно: закрыть rows

	var videos []Video
	for rows.Next() {
		var v Video
		// Сканируем значения из строки результата в поля структуры
		err := rows.Scan(&v.ID, &v.VideoName, &v.FileName, &v.Size)
		if err != nil {
			// Если ошибка при сканировании, прерываем и возвращаем ошибку
			return nil, fmt.Errorf("ошибка сканирования строки: %w", err)
		}
		videos = append(videos, v)
	}

	// Проверяем, не произошла ли ошибка во время итерации по rows
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка итерации по строкам: %w", err)
	}

	return videos, nil
}

// GetVideoByID получает видео по его ID.
// Возвращает указатель на Video и булево значение, указывающее, найдена ли запись.
func (db *DB) GetVideoByID(id int) (*Video, error) {
	querySQL := `SELECT id, video_name, file_name, size FROM videos WHERE id = ?`
	row := db.conn.QueryRow(querySQL, id)

	var v Video
	err := row.Scan(&v.ID, &v.VideoName, &v.FileName, &v.Size)
	if err != nil {

		return nil, fmt.Errorf("ошибка получения видео по ID: %w", err)
	}

	// Видео найдено
	return &v, nil
}

// GetVideoByFileName получает видео по его file_name.
// Возвращает указатель на Video и булево значение, указывающее, найдена ли запись.
func (db *DB) GetVideoByFileName(videoName string) (*Video, error) {
	querySQL := `SELECT id, video_name, file_name, size FROM videos WHERE file_name = ?`
	row := db.conn.QueryRow(querySQL, videoName)

	var v Video
	err := row.Scan(&v.ID, &v.VideoName, &v.FileName, &v.Size)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения видео по file_name: %w", err)
	}

	// Видео найдено
	return &v, nil
}

// UpdateVideo обновляет video_name и/или file_name видео по его ID.
// Пустая строка в новом значении означает, что поле не нужно обновлять.
// Если новое file_name не уникально, будет возвращена ошибка.
func (db *DB) UpdateVideo(id int, newVideoName, newFileName string, size int) error {
	// Строим динамический SQL-запрос
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

	updates = append(updates, "size = ?")
	args = append(args, size)

	// Если ничего не изменилось, просто выходим
	if len(updates) == 1 {
		fmt.Printf("Нет данных для обновления видео с ID %d.\n", id)
		return nil
	}

	// Добавляем ID в аргументы для WHERE
	args = append(args, id)

	querySQL := fmt.Sprintf("UPDATE videos SET %s WHERE id = ?", strings.Join(updates, ", "))
	result, err := db.conn.Exec(querySQL, args...)
	if err != nil {
		return fmt.Errorf("ошибка обновления видео с ID %d: %w", id, err)
	}

	// Проверяем, была ли затронута хотя бы одна строка
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

	// Проверяем, была ли затронута хотя бы одна строка
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

	// Проверяем, была ли затронута хотя бы одна строка
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
