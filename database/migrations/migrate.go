package migrations

import (
	"go-media-center-example/internal/database"
	"go-media-center-example/internal/models"
)

func Migrate() error {
	db := database.GetDB()
	
	// Auto migrate tables
	return db.AutoMigrate(
		&models.User{},
		&models.Folder{},
		&models.Media{},
		&models.Tag{},
	)
}