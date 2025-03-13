package models

import (
	"time"

	"gorm.io/gorm"
)

// Folder represents a folder in the media center
type Folder struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	ParentID    *uint          `json:"parent_id"`
	UserID      uint           `json:"user_id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at,omitempty" gorm:"index"`
	MediaCount  int64          `json:"media_count" gorm:"-"` // Virtual field for media count
}
