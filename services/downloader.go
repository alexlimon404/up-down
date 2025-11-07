package services

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Downloader struct {
	BaseDir    string
	HTTPClient *http.Client
}

func NewDownloader(baseDir string) *Downloader {
	return &Downloader{
		BaseDir: baseDir,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ParseUploadcareURL парсит URL типа https://ucarecdn.com/uuid~count/ или https://ucarecdn.com/uuid/
func (d *Downloader) ParseUploadcareURL(url string) (uuid string, count int, err error) {
	// Пытаемся парсить формат с количеством: https://ucarecdn.com/uuid~count/
	reWithCount := regexp.MustCompile(`https://ucarecdn\.com/([0-9a-f-]+)~(\d+)/?`)
	matches := reWithCount.FindStringSubmatch(url)

	if len(matches) == 3 {
		uuid = matches[1]
		count, err = strconv.Atoi(matches[2])
		if err != nil {
			return "", 0, fmt.Errorf("ошибка парсинга количества файлов: %w", err)
		}
		return uuid, count, nil
	}

	// Пытаемся парсить формат одиночного файла: https://ucarecdn.com/uuid/
	reSingle := regexp.MustCompile(`https://ucarecdn\.com/([0-9a-f-]+)/?`)
	matches = reSingle.FindStringSubmatch(url)

	if len(matches) == 2 {
		uuid = matches[1]
		count = 1
		return uuid, count, nil
	}

	return "", 0, fmt.Errorf("неверный формат URL: %s", url)
}

// DownloadFile скачивает один файл
func (d *Downloader) DownloadFile(url, destPath string) error {
	// Создаём директорию если не существует
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории %s: %w", dir, err)
	}

	// Проверяем, существует ли файл
	if _, err := os.Stat(destPath); err == nil {
		// Файл уже существует, пропускаем
		return nil
	}

	// Скачиваем файл
	resp, err := d.HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("ошибка запроса %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка HTTP %d для %s", resp.StatusCode, url)
	}

	// Создаём временный файл
	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("ошибка создания файла %s: %w", tmpPath, err)
	}
	defer out.Close()

	// Копируем данные
	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("ошибка записи файла %s: %w", tmpPath, err)
	}

	// Переименовываем временный файл
	if err := os.Rename(tmpPath, destPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("ошибка переименования файла: %w", err)
	}

	return nil
}

// DownloadUploadcareFiles скачивает файлы из Uploadcare
func (d *Downloader) DownloadUploadcareFiles(url, destDir, filePrefix string) ([]string, error) {
	if url == "" || url == " " {
		return nil, nil
	}

	url = strings.TrimSpace(url)
	uuid, count, err := d.ParseUploadcareURL(url)
	if err != nil {
		return nil, err
	}

	// Проверяем, это группа файлов или одиночный файл
	isGroup := strings.Contains(url, "~")

	downloadedFiles := make([]string, 0, count)

	for i := 0; i < count; i++ {
		var fileURL string
		if isGroup {
			// Для группы используем формат nth/i/
			fileURL = fmt.Sprintf("https://ucarecdn.com/%s~%d/nth/%d/", uuid, count, i)
		} else {
			// Для одиночного файла используем прямой UUID
			fileURL = fmt.Sprintf("https://ucarecdn.com/%s", uuid)
		}

		// Определяем расширение файла (попробуем скачать и определить)
		ext := d.getFileExtension(fileURL)
		fileName := fmt.Sprintf("%s_%d%s", filePrefix, i+1, ext)
		destPath := filepath.Join(destDir, fileName)

		if err := d.DownloadFile(fileURL, destPath); err != nil {
			return downloadedFiles, fmt.Errorf("ошибка скачивания %s: %w", fileURL, err)
		}

		downloadedFiles = append(downloadedFiles, destPath)
	}

	return downloadedFiles, nil
}

// getFileExtension пытается определить расширение файла
func (d *Downloader) getFileExtension(url string) string {
	resp, err := d.HTTPClient.Head(url)
	if err != nil {
		return ".bin"
	}
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")

	switch {
	case strings.Contains(contentType, "image/jpeg"):
		return ".jpg"
	case strings.Contains(contentType, "image/png"):
		return ".png"
	case strings.Contains(contentType, "image/gif"):
		return ".gif"
	case strings.Contains(contentType, "application/pdf"):
		return ".pdf"
	case strings.Contains(contentType, "image/webp"):
		return ".webp"
	default:
		return ".bin"
	}
}
