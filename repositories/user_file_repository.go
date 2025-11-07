package repositories

import (
	"up-down/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserFileRepository struct {
	db *gorm.DB
}

func NewUserFileRepository(db *gorm.DB) *UserFileRepository {
	return &UserFileRepository{db: db}
}

// Upsert создаёт или обновляет запись о файлах пользователя
func (r *UserFileRepository) Upsert(userID int64, document, address bool) error {
	userFile := models.UserFile{
		UserID:   userID,
		Document: document,
		Address:  address,
	}

	return r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"document", "address", "updated_at"}),
	}).Create(&userFile).Error
}

// GetByUserID получает информацию о файлах пользователя
func (r *UserFileRepository) GetByUserID(userID int64) (*models.UserFile, error) {
	var userFile models.UserFile
	err := r.db.Where("user_id = ?", userID).First(&userFile).Error
	if err != nil {
		return nil, err
	}
	return &userFile, nil
}

// GetAll получает все записи
func (r *UserFileRepository) GetAll() ([]models.UserFile, error) {
	var userFiles []models.UserFile
	err := r.db.Find(&userFiles).Error
	return userFiles, err
}

// GetStats получает статистику
func (r *UserFileRepository) GetStats() (total, withDocument, withAddress, withBoth int64, err error) {
	r.db.Model(&models.UserFile{}).Count(&total)
	r.db.Model(&models.UserFile{}).Where("document = ?", true).Count(&withDocument)
	r.db.Model(&models.UserFile{}).Where("address = ?", true).Count(&withAddress)
	r.db.Model(&models.UserFile{}).Where("document = ? AND address = ?", true, true).Count(&withBoth)
	return
}

// GetPaginated получает записи с пагинацией
func (r *UserFileRepository) GetPaginated(page, perPage int) ([]models.UserFile, int64, error) {
	var userFiles []models.UserFile
	var total int64

	// Подсчитываем общее количество
	r.db.Model(&models.UserFile{}).Count(&total)

	// Получаем записи с пагинацией
	offset := (page - 1) * perPage
	err := r.db.Order("id desc").Offset(offset).Limit(perPage).Find(&userFiles).Error

	return userFiles, total, err
}
