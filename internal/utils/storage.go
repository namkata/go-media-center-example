package utils

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

func SaveFile(file []byte, filename string, storagePath string) (string, error) {
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return "", err
	}

	filePath := filepath.Join(storagePath, filename)
	if err := os.WriteFile(filePath, file, 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

func ProcessImage(filePath string) error {
	img, err := imaging.Open(filePath)
	if err != nil {
		return err
	}

	// Resize if image is too large
	if img.Bounds().Dx() > 1920 {
		img = imaging.Resize(img, 1920, 0, imaging.Lanczos)
	}

	// Save processed image
	return imaging.Save(img, filePath)
}

func GetFileType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif":
		return "image"
	case ".mp4", ".mov", ".avi":
		return "video"
	default:
		return "other"
	}
}
