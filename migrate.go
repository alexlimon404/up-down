package main

import (
	"fmt"
	"log"
	"up-down/config"
	"up-down/database"
	"up-down/models"
)

func main() {
	// Загрузка конфигурации
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Подключение ко второй базе данных через GORM
	db, err := database.NewGorm(&cfg.Database2)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}

	fmt.Printf("✓ Подключено к БД: %s\n", cfg.Database2.DBName)

	// Автоматическая миграция
	err = db.AutoMigrate(&models.UserFile{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	fmt.Println("✓ Миграция успешно применена!")
	fmt.Println("✓ Таблица user_files создана через GORM")
}
