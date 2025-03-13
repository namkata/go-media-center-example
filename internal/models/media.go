package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"go-media-center-example/internal/database"

	"gorm.io/gorm"
)

// Media represents a media file in the system
type Media struct {
	ID        string `gorm:"primarykey"`
	UserID    uint
	FolderID  *string
	Filename  string
	Path      string
	MimeType  string
	Size      int64
	Metadata  json.RawMessage `gorm:"type:jsonb"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Tags      []Tag          `gorm:"many2many:media_tags;"`
}

// JSON is a custom type for handling JSON data in the database
type JSON map[string]interface{}

// Scan implements the sql.Scanner interface
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = JSON{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONB value")
	}

	var result map[string]interface{}
	err := json.Unmarshal(bytes, &result)
	if err != nil {
		return err
	}
	*j = JSON(result)
	return nil
}

// Value implements the driver.Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

type Tag struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `json:"name" gorm:"unique"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
	Media     []Media        `gorm:"many2many:media_tags;"`
}

// BeforeCreate hook to ensure Metadata is properly handled
func (m *Media) BeforeCreate(tx *gorm.DB) error {
	if m.Metadata == nil {
		m.Metadata = json.RawMessage("{}")
	}
	return nil
}

// GetMediaByID retrieves a media record by its ID
func GetMediaByID(id string) (*Media, error) {
	var media Media
	db := database.GetDB()
	if db == nil {
		return nil, errors.New("database connection not initialized")
	}

	result := db.Model(&Media{}).First(&media, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &media, nil
}

// TableName specifies the table name for the Media model
func (Media) TableName() string {
	return "media"
}
