package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"up-down/config"
	"up-down/database"
	"up-down/models"
	"up-down/repositories"
	"up-down/services"
)

type WebHandler struct {
	userFileRepo    *repositories.UserFileRepository
	db              *database.DB
	cfg             *config.Config
	templates       *template.Template
	downloadManager *services.DownloadManager
}

func NewWebHandler(userFileRepo *repositories.UserFileRepository, db *database.DB, cfg *config.Config, downloadManager *services.DownloadManager) *WebHandler {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	return &WebHandler{
		userFileRepo:    userFileRepo,
		db:              db,
		cfg:             cfg,
		templates:       tmpl,
		downloadManager: downloadManager,
	}
}

// IndexHandler отображает главную страницу
func (h *WebHandler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.templates.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetUsersHandler возвращает список пользователей с пагинацией
func (h *WebHandler) GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем параметры пагинации
	page := 1
	perPage := 20
	sortOrder := "DESC" // По умолчанию сортировка по убыванию

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if perPageStr := r.URL.Query().Get("per_page"); perPageStr != "" {
		if pp, err := strconv.Atoi(perPageStr); err == nil && pp > 0 && pp <= 100 {
			perPage = pp
		}
	}

	// Получаем параметр сортировки
	if sortOrderStr := r.URL.Query().Get("sort_order"); sortOrderStr != "" {
		sortOrderUpper := strings.ToUpper(sortOrderStr)
		if sortOrderUpper == "ASC" || sortOrderUpper == "DESC" {
			sortOrder = sortOrderUpper
		}
	}

	// Подсчитываем общее количество пользователей с файлами в основной БД
	var total int64
	err := h.db.QueryRow(`
		SELECT COUNT(*)
		FROM users
		WHERE (document_files IS NOT NULL AND document_files != '')
		   OR (address_files IS NOT NULL AND address_files != '')
	`).Scan(&total)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем пользователей из основной БД с пагинацией
	offset := (page - 1) * perPage
	query := fmt.Sprintf(`
		SELECT id, citizenship_id, document_files, address_files
		FROM users
		WHERE (document_files IS NOT NULL AND document_files != '')
		   OR (address_files IS NOT NULL AND address_files != '')
		ORDER BY id %s
		LIMIT $1 OFFSET $2
	`, sortOrder)

	rows, err := h.db.Query(query, perPage, offset)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	views := make([]models.UserFileView, 0)
	for rows.Next() {
		var userID int64
		var citizenshipID sql.NullString
		var documentFiles sql.NullString
		var addressFiles sql.NullString

		if err := rows.Scan(&userID, &citizenshipID, &documentFiles, &addressFiles); err != nil {
			continue
		}

		view := models.UserFileView{
			UserID:        userID,
			CitizenshipID: citizenshipID.String,
			DocumentFiles: documentFiles.String,
			AddressFiles:  addressFiles.String,
			Document:      false,
			Address:       false,
		}

		// Проверяем статус в user_files
		userFile, err := h.userFileRepo.GetByUserID(userID)
		if err == nil && userFile != nil {
			view.Document = userFile.Document
			view.Address = userFile.Address
		}

		views = append(views, view)
	}

	// Формируем ответ
	totalPages := int(math.Ceil(float64(total) / float64(perPage)))
	response := models.PaginatedResponse{
		Data:       views,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
		SortOrder:  sortOrder,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DownloadHandler обрабатывает запрос на скачивание файлов пользователя
func (h *WebHandler) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	// Получаем данные о файлах пользователя
	var citizenshipID sql.NullString
	query := `SELECT citizenship_id FROM users WHERE id = $1`
	err = h.db.QueryRow(query, userID).Scan(&citizenshipID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	// Формируем путь к файлам
	if !citizenshipID.Valid || citizenshipID.String == "" {
		http.Error(w, "citizenship_id not found", http.StatusNotFound)
		return
	}

	// Возвращаем JSON с информацией о пути к файлам
	response := map[string]string{
		"user_id":        userIDStr,
		"citizenship_id": citizenshipID.String,
		"path":           h.cfg.Download.Dir + "/" + citizenshipID.String + "/user_" + userIDStr,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DownloadUserFilesHandler скачивает файлы конкретного пользователя
func (h *WebHandler) DownloadUserFilesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	// Получаем данные пользователя
	var citizenshipID sql.NullString
	var documentFiles sql.NullString
	var addressFiles sql.NullString
	var phone sql.NullString
	var email sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var patronymic sql.NullString
	var documentNumber sql.NullString

	query := `SELECT citizenship_id, document_files, address_files, phone, email, first_name, last_name, patronymic, document_number FROM users WHERE id = $1`
	err = h.db.QueryRow(query, userID).Scan(&citizenshipID, &documentFiles, &addressFiles, &phone, &email, &firstName, &lastName, &patronymic, &documentNumber)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Пользователь не найден",
		})
		return
	}

	// Проверяем citizenship_id
	if !citizenshipID.Valid || citizenshipID.String == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "У пользователя нет citizenship_id",
		})
		return
	}

	// Формируем путь
	userDir := fmt.Sprintf("%s/%s/user_%d", h.cfg.Download.Dir, citizenshipID.String, userID)

	downloadedFiles := make([]string, 0)
	errors := make([]string, 0)
	documentSuccess := false
	addressSuccess := false

	// Проверяем, есть ли вообще файлы для скачивания
	if (!documentFiles.Valid || documentFiles.String == "") && (!addressFiles.Valid || addressFiles.String == "") {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "У пользователя нет файлов для скачивания",
		})
		return
	}

	// Скачиваем document_files
	if documentFiles.Valid && documentFiles.String != "" {
		docDir := userDir + "/documents"
		files, err := services.NewDownloader(h.cfg.Download.Dir).DownloadUploadcareFiles(documentFiles.String, docDir, "document")
		if err != nil {
			errors = append(errors, fmt.Sprintf("Document files: %v", err))
		} else {
			downloadedFiles = append(downloadedFiles, files...)
			documentSuccess = true
		}
	}

	// Скачиваем address_files
	if addressFiles.Valid && addressFiles.String != "" {
		addrDir := userDir + "/address"
		files, err := services.NewDownloader(h.cfg.Download.Dir).DownloadUploadcareFiles(addressFiles.String, addrDir, "address")
		if err != nil {
			errors = append(errors, fmt.Sprintf("Address files: %v", err))
		} else {
			downloadedFiles = append(downloadedFiles, files...)
			addressSuccess = true
		}
	}

	// Сохраняем статус в БД
	if documentSuccess || addressSuccess {
		h.userFileRepo.Upsert(userID, documentSuccess, addressSuccess)

		// Создаём файл info.txt с информацией о пользователе
		phoneStr := "N/A"
		if phone.Valid && phone.String != "" {
			phoneStr = phone.String
		}

		emailStr := "N/A"
		if email.Valid && email.String != "" {
			emailStr = email.String
		}

		firstNameStr := "N/A"
		if firstName.Valid && firstName.String != "" {
			firstNameStr = firstName.String
		}

		lastNameStr := "N/A"
		if lastName.Valid && lastName.String != "" {
			lastNameStr = lastName.String
		}

		patronymicStr := "N/A"
		if patronymic.Valid && patronymic.String != "" {
			patronymicStr = patronymic.String
		}

		documentNumberStr := "N/A"
		if documentNumber.Valid && documentNumber.String != "" {
			documentNumberStr = documentNumber.String
		}

		infoContent := fmt.Sprintf("phone: %s\nemail: %s\nfirst_name: %s\nlast_name: %s\npatronymic: %s\ndocument_number: %s\n",
			phoneStr, emailStr, firstNameStr, lastNameStr, patronymicStr, documentNumberStr)
		infoFilePath := userDir + "/info.txt"

		if err := os.WriteFile(infoFilePath, []byte(infoContent), 0644); err != nil {
			errors = append(errors, fmt.Sprintf("Info file: %v", err))
		}
	}

	// Формируем ответ
	response := map[string]interface{}{
		"success":          len(downloadedFiles) > 0,
		"user_id":          userID,
		"citizenship_id":   citizenshipID.String,
		"path":             userDir,
		"files_downloaded": len(downloadedFiles),
		"document_success": documentSuccess,
		"address_success":  addressSuccess,
	}

	if len(errors) > 0 {
		response["errors"] = errors
		if len(downloadedFiles) == 0 {
			response["error"] = fmt.Sprintf("Ошибка скачивания файлов: %s", strings.Join(errors, "; "))
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// StartDownloadHandler запускает процесс скачивания
func (h *WebHandler) StartDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.downloadManager.Start()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "started",
	})
}

// StopDownloadHandler останавливает процесс скачивания
func (h *WebHandler) StopDownloadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	h.downloadManager.Stop()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "stopped",
	})
}

// GetProgressHandler возвращает текущий прогресс скачивания
func (h *WebHandler) GetProgressHandler(w http.ResponseWriter, r *http.Request) {
	status, stats, duration := h.downloadManager.GetStatus()

	response := map[string]interface{}{
		"status":           string(status),
		"total_users":      stats.TotalUsers,
		"processed_users":  stats.ProcessedUsers,
		"successful_users": stats.SuccessfulUsers,
		"failed_users":     stats.FailedUsers,
		"total_files":      stats.TotalFiles,
		"successful_files": stats.SuccessfulFiles,
		"failed_files":     stats.FailedFiles,
		"skipped_users":    stats.SkippedUsers,
		"duration_seconds": duration.Seconds(),
	}

	if stats.TotalUsers > 0 {
		response["progress_percent"] = float64(stats.ProcessedUsers) / float64(stats.TotalUsers) * 100
	} else {
		response["progress_percent"] = 0.0
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetDownloadStatsHandler возвращает статистику скачивания
func (h *WebHandler) GetDownloadStatsHandler(w http.ResponseWriter, r *http.Request) {
	// Получаем все статусы из второй БД одним запросом
	userFilesMap, err := h.userFileRepo.GetAllAsMap()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем всех пользователей с файлами из первой БД
	rows, err := h.db.Query(`
		SELECT id,
		       (document_files IS NOT NULL AND document_files != '') as has_doc,
		       (address_files IS NOT NULL AND address_files != '') as has_addr
		FROM users
		WHERE (document_files IS NOT NULL AND document_files != '')
		   OR (address_files IS NOT NULL AND address_files != '')
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var totalUsersWithFiles int64
	var fullyDownloaded int64
	var partiallyDownloaded int64
	var notDownloaded int64

	// Проходим по каждому пользователю и проверяем его статус
	for rows.Next() {
		var userID int64
		var hasDoc, hasAddr bool

		if err := rows.Scan(&userID, &hasDoc, &hasAddr); err != nil {
			continue
		}

		totalUsersWithFiles++

		// Ищем статус в map (уже загружен из БД)
		userFile, exists := userFilesMap[userID]

		if !exists {
			// Нет записи в user_files - файлы не скачаны
			notDownloaded++
			continue
		}

		// Проверяем, все ли требуемые файлы скачаны
		docOk := !hasDoc || userFile.Document
		addrOk := !hasAddr || userFile.Address

		if docOk && addrOk {
			// Все требуемые файлы скачаны
			fullyDownloaded++
		} else if userFile.Document || userFile.Address {
			// Хотя бы один тип файлов скачан, но не все
			partiallyDownloaded++
		} else {
			// Запись есть, но файлы не скачаны
			notDownloaded++
		}
	}

	response := map[string]interface{}{
		"total_users":          totalUsersWithFiles,
		"fully_downloaded":     fullyDownloaded,
		"partially_downloaded": partiallyDownloaded,
		"not_downloaded":       notDownloaded,
		"remaining":            totalUsersWithFiles - fullyDownloaded,
		"progress_percent":     0.0,
	}

	if totalUsersWithFiles > 0 {
		response["progress_percent"] = float64(fullyDownloaded) / float64(totalUsersWithFiles) * 100
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
