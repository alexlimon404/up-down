package services

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
	"up-down/config"
	"up-down/database"
	"up-down/models"
	"up-down/repositories"
)

type DownloadStatus string

const (
	StatusIdle      DownloadStatus = "idle"
	StatusRunning   DownloadStatus = "running"
	StatusPaused    DownloadStatus = "paused"
	StatusCompleted DownloadStatus = "completed"
	StatusFailed    DownloadStatus = "failed"
)

type Stats struct {
	TotalUsers      int64
	ProcessedUsers  int64
	SuccessfulUsers int64
	FailedUsers     int64
	TotalFiles      int64
	SuccessfulFiles int64
	FailedFiles     int64
	SkippedUsers    int64
}

type DownloadManager struct {
	cfg          *config.Config
	db           *database.DB
	userFileRepo *repositories.UserFileRepository
	downloader   *Downloader

	status    DownloadStatus
	stats     *Stats
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mutex     sync.RWMutex
	startTime time.Time
	endTime   time.Time
}

func NewDownloadManager(cfg *config.Config, db *database.DB, userFileRepo *repositories.UserFileRepository) *DownloadManager {
	return &DownloadManager{
		cfg:          cfg,
		db:           db,
		userFileRepo: userFileRepo,
		downloader:   NewDownloader(cfg.Download.Dir),
		status:       StatusIdle,
		stats:        &Stats{},
	}
}

// Start запускает процесс скачивания
func (dm *DownloadManager) Start() error {
	dm.mutex.Lock()
	if dm.status == StatusRunning {
		dm.mutex.Unlock()
		return fmt.Errorf("скачивание уже запущено")
	}

	dm.status = StatusRunning
	dm.stats = &Stats{} // Сбрасываем статистику
	dm.ctx, dm.cancel = context.WithCancel(context.Background())
	dm.startTime = time.Now()
	dm.mutex.Unlock()

	go dm.run()
	return nil
}

// Stop останавливает процесс скачивания
func (dm *DownloadManager) Stop() {
	dm.mutex.Lock()
	if dm.status != StatusRunning {
		dm.mutex.Unlock()
		return
	}
	dm.mutex.Unlock()

	if dm.cancel != nil {
		dm.cancel()
	}
	dm.wg.Wait()

	dm.mutex.Lock()
	dm.status = StatusIdle
	dm.endTime = time.Now()
	dm.mutex.Unlock()
}

// GetStatus возвращает текущий статус
func (dm *DownloadManager) GetStatus() (DownloadStatus, *Stats, time.Duration) {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()

	statsCopy := &Stats{
		TotalUsers:      atomic.LoadInt64(&dm.stats.TotalUsers),
		ProcessedUsers:  atomic.LoadInt64(&dm.stats.ProcessedUsers),
		SuccessfulUsers: atomic.LoadInt64(&dm.stats.SuccessfulUsers),
		FailedUsers:     atomic.LoadInt64(&dm.stats.FailedUsers),
		TotalFiles:      atomic.LoadInt64(&dm.stats.TotalFiles),
		SuccessfulFiles: atomic.LoadInt64(&dm.stats.SuccessfulFiles),
		FailedFiles:     atomic.LoadInt64(&dm.stats.FailedFiles),
		SkippedUsers:    atomic.LoadInt64(&dm.stats.SkippedUsers),
	}

	var duration time.Duration
	if dm.status == StatusRunning {
		duration = time.Since(dm.startTime)
	} else if !dm.endTime.IsZero() {
		duration = dm.endTime.Sub(dm.startTime)
	}

	return dm.status, statsCopy, duration
}

// run выполняет процесс скачивания
func (dm *DownloadManager) run() {
	// Подсчитываем общее количество пользователей
	err := dm.db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE (document_files IS NOT NULL AND document_files != '')
		   OR (address_files IS NOT NULL AND address_files != '')
	`).Scan(&dm.stats.TotalUsers)
	if err != nil {
		log.Printf("Ошибка подсчёта пользователей: %v", err)
		dm.mutex.Lock()
		dm.status = StatusFailed
		dm.mutex.Unlock()
		return
	}

	// Канал для пользователей
	usersChan := make(chan *models.User, dm.cfg.Download.BatchSize)

	// Запускаем воркеры
	for i := 0; i < dm.cfg.Download.Workers; i++ {
		dm.wg.Add(1)
		go dm.worker(i+1, usersChan)
	}

	// Читаем пользователей из БД
	err = dm.fetchUsers(usersChan)
	if err != nil {
		log.Printf("Ошибка чтения пользователей: %v", err)
	}

	// Закрываем канал пользователей
	close(usersChan)

	// Ждём завершения всех воркеров
	dm.wg.Wait()

	dm.mutex.Lock()
	if dm.ctx.Err() != nil {
		dm.status = StatusIdle
	} else {
		dm.status = StatusCompleted
	}
	dm.endTime = time.Now()
	dm.mutex.Unlock()
}

func (dm *DownloadManager) fetchUsers(usersChan chan<- *models.User) error {
	offset := 0

	for {
		select {
		case <-dm.ctx.Done():
			return dm.ctx.Err()
		default:
		}

		query := `
			SELECT id, citizenship_id, document_files, address_files
			FROM users
			WHERE (document_files IS NOT NULL AND document_files != '')
			   OR (address_files IS NOT NULL AND address_files != '')
			ORDER BY id
			LIMIT $1 OFFSET $2
		`

		rows, err := dm.db.Query(query, dm.cfg.Download.BatchSize, offset)
		if err != nil {
			return fmt.Errorf("ошибка запроса: %w", err)
		}

		count := 0
		for rows.Next() {
			user := &models.User{}
			if err := rows.Scan(&user.ID, &user.CitizenshipID, &user.DocumentFiles, &user.AddressFiles); err != nil {
				log.Printf("Ошибка сканирования пользователя: %v", err)
				continue
			}

			select {
			case usersChan <- user:
				count++
			case <-dm.ctx.Done():
				rows.Close()
				return dm.ctx.Err()
			}
		}
		rows.Close()

		if count == 0 {
			break
		}

		offset += dm.cfg.Download.BatchSize
	}

	return nil
}

func (dm *DownloadManager) worker(id int, usersChan <-chan *models.User) {
	defer dm.wg.Done()

	for {
		select {
		case <-dm.ctx.Done():
			return
		case user, ok := <-usersChan:
			if !ok {
				return
			}

			atomic.AddInt64(&dm.stats.ProcessedUsers, 1)

			// Проверяем citizenship_id
			if !user.CitizenshipID.Valid || user.CitizenshipID.String == "" {
				atomic.AddInt64(&dm.stats.SkippedUsers, 1)
				continue
			}

			// Проверяем статус уже скачанных файлов
			existingStatus, err := dm.userFileRepo.GetByUserID(user.ID)
			documentAlreadyDownloaded := false
			addressAlreadyDownloaded := false

			if err == nil && existingStatus != nil {
				documentAlreadyDownloaded = existingStatus.Document
				addressAlreadyDownloaded = existingStatus.Address
			}

			// Если оба типа файлов уже скачаны, пропускаем пользователя
			needDownloadDocument := user.DocumentFiles.Valid && user.DocumentFiles.String != "" && !documentAlreadyDownloaded
			needDownloadAddress := user.AddressFiles.Valid && user.AddressFiles.String != "" && !addressAlreadyDownloaded

			if !needDownloadDocument && !needDownloadAddress {
				atomic.AddInt64(&dm.stats.SkippedUsers, 1)
				log.Printf("[Worker %d] ⏭️  user_id: %d - файлы уже скачаны, пропускаем", id, user.ID)
				continue
			}

			// Создаём директорию для пользователя
			userDir := filepath.Join(dm.downloader.BaseDir, user.CitizenshipID.String, fmt.Sprintf("user_%d", user.ID))

			hasErrors := false
			documentSuccess := documentAlreadyDownloaded // Сохраняем предыдущий статус
			addressSuccess := addressAlreadyDownloaded   // Сохраняем предыдущий статус

			// Скачиваем document_files только если еще не скачаны
			if needDownloadDocument {
				docDir := filepath.Join(userDir, "documents")
				files, err := dm.downloader.DownloadUploadcareFiles(user.DocumentFiles.String, docDir, "document")
				if err != nil {
					log.Printf("[Worker %d] Ошибка скачивания документов пользователя %d: %v", id, user.ID, err)
					hasErrors = true
					atomic.AddInt64(&dm.stats.FailedFiles, 1)
				} else {
					atomic.AddInt64(&dm.stats.TotalFiles, int64(len(files)))
					atomic.AddInt64(&dm.stats.SuccessfulFiles, int64(len(files)))
					documentSuccess = true
					log.Printf("[Worker %d] 📄 user_id: %d - скачано %d документов", id, user.ID, len(files))
				}
			} else if documentAlreadyDownloaded {
				log.Printf("[Worker %d] ⏭️  user_id: %d - документы уже скачаны ранее", id, user.ID)
			}

			// Скачиваем address_files только если еще не скачаны
			if needDownloadAddress {
				addrDir := filepath.Join(userDir, "address")
				files, err := dm.downloader.DownloadUploadcareFiles(user.AddressFiles.String, addrDir, "address")
				if err != nil {
					log.Printf("[Worker %d] Ошибка скачивания адресных файлов пользователя %d: %v", id, user.ID, err)
					hasErrors = true
					atomic.AddInt64(&dm.stats.FailedFiles, 1)
				} else {
					atomic.AddInt64(&dm.stats.TotalFiles, int64(len(files)))
					atomic.AddInt64(&dm.stats.SuccessfulFiles, int64(len(files)))
					addressSuccess = true
					log.Printf("[Worker %d] 🏠 user_id: %d - скачано %d адресных файлов", id, user.ID, len(files))
				}
			} else if addressAlreadyDownloaded {
				log.Printf("[Worker %d] ⏭️  user_id: %d - адресные файлы уже скачаны ранее", id, user.ID)
			}

			// Записываем статус в базу данных
			if documentSuccess || addressSuccess {
				if err := dm.userFileRepo.Upsert(user.ID, documentSuccess, addressSuccess); err != nil {
					log.Printf("[Worker %d] Ошибка записи статуса для пользователя %d: %v", id, user.ID, err)
				}
			}

			if hasErrors {
				atomic.AddInt64(&dm.stats.FailedUsers, 1)
				log.Printf("[Worker %d] ❌ user_id: %d - скачивание завершено с ошибками", id, user.ID)
			} else {
				atomic.AddInt64(&dm.stats.SuccessfulUsers, 1)
				log.Printf("[Worker %d] ✅ user_id: %d - обработка завершена успешно (документы: %v, адрес: %v)", id, user.ID, documentSuccess, addressSuccess)
			}
		}
	}
}
