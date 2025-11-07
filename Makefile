.PHONY: help run migrate build clean

help: ## Показать справку
	@echo "Доступные команды:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

run: ## Запустить приложение (веб-интерфейс + скачивание)
	go run main.go

migrate: ## Запустить миграции
	go run migrate.go

build: ## Собрать бинарный файл
	go build -o up-down main.go

clean: ## Очистить собранные файлы
	rm -f up-down
	go clean

install: ## Установить зависимости
	go mod download
	go mod tidy

.DEFAULT_GOAL := help
