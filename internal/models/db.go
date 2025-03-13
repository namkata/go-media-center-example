package models

import (
	"fmt"
	"log"

	"go-media-center-example/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// InitDB initializes the database connection
func InitDB() error {
	cfg := config.GetConfig()
	dsn := cfg.Database.DSN()

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	// Auto-migrate models
	if err := DB.AutoMigrate(
		&Media{},
		&Folder{},
		&User{},
		&Tag{},
	); err != nil {
		return fmt.Errorf("failed to migrate database: %v", err)
	}

	log.Println("Database connection established")
	return nil
}
