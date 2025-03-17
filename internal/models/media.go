package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"log"
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

// MarshalJSON implements the json.Marshaler interface
func (j JSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}(j))
}

// UnmarshalJSON implements the json.Unmarshaler interface
func (j *JSON) UnmarshalJSON(data []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}
	*j = JSON(m)
	return nil
}

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

	log.Printf("Fetching media with ID: %s", id)
	result := db.Table(media.TableName()).Where("id = ?", id).First(&media)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Println("No media found for the given ID")
		return nil, nil // No error, but media is not found
	} else if result.Error != nil {
		log.Printf("Error fetching media: %v", result.Error)
		return nil, result.Error
	}

	return &media, nil
}

// TableName specifies the table name for the Media model
func (Media) TableName() string {
	return "media"
}
