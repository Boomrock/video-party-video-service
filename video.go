package main

import (
	"fmt"
	"net/http"
	"time"
	"video/database"
	"video/handlers/video"
	"video/logger"
	"video/streamer"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

// 1. Создание комнаты
// 2. Закачка на сервер
// 3. Синхронизация видео

func main() {
	sqllite, err := database.New("./sqlite.db")
	if err != nil {
		fmt.Printf(fmt.Errorf("база данных не открылась: %w", err).Error())
		return
	}
	err = sqllite.CreateTable()
	if err != nil {
		fmt.Printf(fmt.Errorf("база данных не создалась: %w", err).Error())
		return
	}
	streamer := streamer.FileStreamer{}
	// Регистрируем обработчик

	router := chi.NewRouter()
	router.Use(logger.Middlerware)                   // Логирование запросов
	router.Use(middleware.Recoverer)                 // Восстановление после паники
	router.Use(middleware.Timeout(30 * time.Second)) // Таймаут на обработку

	router.Get("/video", video.Sender(&streamer, sqllite))
	router.Get("/allVideo", video.GetAllVideo(sqllite))
	router.Post("/videoUpload", video.Upload(sqllite))

	fmt.Println("Сервер запущен на http://localhost:8080")
	fmt.Println("Открой в браузере: http://localhost:8080/video")
	http.ListenAndServe(":8080", router)
}
