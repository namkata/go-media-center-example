package models

import (
	"time"
)

// This file contains type definitions to help Swagger generate documentation

// SwaggerJSON is a type for handling JSON data in Swagger documentation
type SwaggerJSON map[string]interface{}

// SwaggerMedia is a simplified version of Media for Swagger documentation
// @Description Media file information
type SwaggerMedia struct {
	ID        string      `json:"id" example:"3f8d9a7c-5e4b-4b3a-8e1d-7f6b5c4d3a2b"`
	UserID    uint        `json:"user_id" example:"1"`
	FolderID  *string     `json:"folder_id,omitempty" example:"folder123"`
	Filename  string      `json:"filename" example:"vacation.jpg"`
	Path      string      `json:"path" example:"uploads/vacation.jpg"`
	MimeType  string      `json:"mime_type" example:"image/jpeg"`
	Size      int64       `json:"size" example:"1024000"`
	Metadata  SwaggerJSON `json:"metadata,omitempty" swaggertype:"object"`
	Tags      []Tag       `json:"tags,omitempty"`
	CreatedAt time.Time   `json:"created_at" example:"2023-01-01T12:00:00Z"`
	UpdatedAt time.Time   `json:"updated_at" example:"2023-01-01T12:00:00Z"`
}
