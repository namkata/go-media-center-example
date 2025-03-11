package models

import (
	"gorm.io/gorm"
)

type Folder struct {
	gorm.Model
	Name     string   `json:"name" gorm:"not null"`
	ParentID *uint    `json:"parent_id" gorm:"index"`
	UserID   uint     `json:"user_id" gorm:"not null;index"`
	Media    []Media  `json:"media,omitempty"`
	MediaCount int64    `json:"media_count" gorm:"-"`
}