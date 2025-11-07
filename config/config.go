package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Database  DatabaseConfig
	Database2 DatabaseConfig
	Server    ServerConfig
	Download  DownloadConfig
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ServerConfig struct {
	Port string
}

type DownloadConfig struct {
	Dir       string
	BatchSize int
	Workers   int
}

func Load() (*Config, error) {
	// Загружаем переменные окружения из .env файла
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("ошибка загрузки .env файла: %w", err)
	}

	// Парсим порт первой базы данных
	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат DB_PORT: %w", err)
	}

	// Парсим порт второй базы данных
	db2Port, err := strconv.Atoi(getEnv("DB2_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат DB2_PORT: %w", err)
	}

	// Парсим настройки скачивания
	batchSize, err := strconv.Atoi(getEnv("BATCH_SIZE", "100"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат BATCH_SIZE: %w", err)
	}

	workers, err := strconv.Atoi(getEnv("WORKERS", "5"))
	if err != nil {
		return nil, fmt.Errorf("неверный формат WORKERS: %w", err)
	}

	config := &Config{
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "updown"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Database2: DatabaseConfig{
			Host:     getEnv("DB2_HOST", "localhost"),
			Port:     db2Port,
			User:     getEnv("DB2_USER", "postgres"),
			Password: getEnv("DB2_PASSWORD", "postgres"),
			DBName:   getEnv("DB2_NAME", "up-down"),
			SSLMode:  getEnv("DB2_SSLMODE", "disable"),
		},
		Server: ServerConfig{
			Port: getEnv("SERVER_PORT", "8080"),
		},
		Download: DownloadConfig{
			Dir:       getEnv("DOWNLOAD_DIR", "./downloads"),
			BatchSize: batchSize,
			Workers:   workers,
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *DatabaseConfig) ConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}
