package models

import "time"

type UserFile struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	UserID    int64     `gorm:"uniqueIndex;not null" json:"user_id"`
	Document  bool      `gorm:"default:false" json:"document"`
	Address   bool      `gorm:"default:false" json:"address"`
}

func (UserFile) TableName() string {
	return "user_files"
}
