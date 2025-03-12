package utils

import (
	"bytes"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

// ProcessImageWithSize resizes an image to the specified dimensions
func ProcessImageWithSize(reader io.Reader, width, height int, fit string) ([]byte, error) {
	// Decode image
	src, err := imaging.Decode(reader)
	if err != nil {
		return nil, err
	}

	// Calculate target dimensions while maintaining aspect ratio
	srcBounds := src.Bounds()
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()
	targetWidth := width
	targetHeight := height

	if targetWidth == 0 {
		// Calculate width based on height while maintaining aspect ratio
		targetWidth = int(float64(srcWidth) * float64(targetHeight) / float64(srcHeight))
	} else if targetHeight == 0 {
		// Calculate height based on width while maintaining aspect ratio
		targetHeight = int(float64(srcHeight) * float64(targetWidth) / float64(srcWidth))
	}

	// Resize based on fit mode
	var resized *image.NRGBA
	switch fit {
	case "cover":
		// Resize and crop to fill the target dimensions
		resized = imaging.Fill(src, targetWidth, targetHeight, imaging.Center, imaging.Lanczos)
	case "fill":
		// Stretch to fill the target dimensions
		resized = imaging.Resize(src, targetWidth, targetHeight, imaging.Lanczos)
	default: // "contain"
		// Resize to fit within the target dimensions while maintaining aspect ratio
		resized = imaging.Fit(src, targetWidth, targetHeight, imaging.Lanczos)
	}

	// Encode the resized image
	buf := new(bytes.Buffer)
	err = imaging.Encode(buf, resized, imaging.PNG)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

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
