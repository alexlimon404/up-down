package main

import (
	"fmt"
	"log"
	"net/http"
	"up-down/config"
	"up-down/database"
	"up-down/handlers"
	"up-down/models"
	"up-down/repositories"
	"up-down/services"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Подключение к первой БД (источник данных)
	db, err := database.New(&cfg.Database)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Подключение ко второй БД через GORM (для логирования статуса)
	db2, err := database.NewGorm(&cfg.Database2)
	if err != nil {
		log.Fatalf("Ошибка подключения ко второй базе данных: %v", err)
	}

	// Автоматическая миграция
	if err := db2.AutoMigrate(&models.UserFile{}); err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	// Создаём репозиторий
	userFileRepo := repositories.NewUserFileRepository(db2)

	// Создаём менеджер скачивания
	downloadManager := services.NewDownloadManager(cfg, db, userFileRepo)

	// Создаём handler
	webHandler := handlers.NewWebHandler(userFileRepo, db, cfg, downloadManager)

	// Настройка маршрутов
	http.HandleFunc("/", webHandler.IndexHandler)
	http.HandleFunc("/api/users", webHandler.GetUsersHandler)
	http.HandleFunc("/api/download", webHandler.DownloadHandler)
	http.HandleFunc("/api/download/user", webHandler.DownloadUserFilesHandler)
	http.HandleFunc("/api/download/start", webHandler.StartDownloadHandler)
	http.HandleFunc("/api/download/stop", webHandler.StopDownloadHandler)
	http.HandleFunc("/api/download/progress", webHandler.GetProgressHandler)
	http.HandleFunc("/api/download/stats", webHandler.GetDownloadStatsHandler)

	// Статические файлы
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Запуск сервера
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	fmt.Println("╔══════════════════════════════════════════════════╗")
	fmt.Println("║     Up-Down - File Download Manager             ║")
	fmt.Println("╚══════════════════════════════════════════════════╝")
	fmt.Printf("\n✓ База данных (источник): %s\n", cfg.Database.DBName)
	fmt.Printf("✓ База данных (логирование): %s\n", cfg.Database2.DBName)
	fmt.Printf("✓ Веб-сервер запущен на: http://localhost%s\n\n", addr)
	fmt.Println("📊 Откройте браузер и перейдите по адресу выше")
	fmt.Println("🚀 Нажмите 'Запустить скачивание' в веб-интерфейсе")
	fmt.Println("\nНажмите Ctrl+C для остановки сервера")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Ошибка запуска веб-сервера: %v", err)
	}
}
