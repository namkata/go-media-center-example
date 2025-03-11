package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
)

type Media struct {
	gorm.Model
	Name     string `json:"name" gorm:"not null"`
	Type     string `json:"type" gorm:"not null"`
	Size     int64  `json:"size" gorm:"not null"`
	URL      string `json:"url" gorm:"not null"`
	FolderID *uint  `json:"folder_id" gorm:"index"`
	UserID   uint   `json:"user_id" gorm:"not null;index"`
	Metadata JSON   `json:"metadata" gorm:"type:jsonb"`
	Tags     []Tag  `json:"tags" gorm:"many2many:media_tags;"`
}

type JSON map[string]interface{}

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}

	var result map[string]interface{}
	err := json.Unmarshal(bytes, &result)
	*j = JSON(result)
	return err
}

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

type Tag struct {
	gorm.Model
	Name  string  `json:"name" gorm:"unique"`
	Media []Media `json:"media" gorm:"many2many:media_tags;"`
}

// BeforeCreate hook to ensure Tags are properly handled
func (m *Media) BeforeCreate(tx *gorm.DB) error {
	if m.Metadata == nil {
		m.Metadata = make(JSON)
	}
	return nil
}
