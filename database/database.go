package database

import (
	"database/sql"
	"fmt"
	"up-down/config"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DB struct {
	*sql.DB
}

func New(cfg *config.DatabaseConfig) (*DB, error) {
	connStr := cfg.ConnectionString()

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть подключение к базе данных: %w", err)
	}

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
	}

	// Настройка пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

// NewGorm создаёт подключение с использованием GORM
func NewGorm(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	dsn := cfg.ConnectionString()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к базе данных через GORM: %w", err)
	}

	// Получаем базовое подключение для настройки пула
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("не удалось получить sql.DB: %w", err)
	}

	// Настройка пула соединений
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)

	return db, nil
}
