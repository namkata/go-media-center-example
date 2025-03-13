package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"go-media-center-example/internal/database"
	"go-media-center-example/internal/models"
	"go-media-center-example/internal/utils"
)

// BatchOperation represents a batch operation request
type BatchOperation struct {
	MediaID         uint                        `json:"media_id"`
	Transformations utils.TransformationOptions `json:"transformations"`
}

// HandleBatchOperation handles batch operations on media files
func HandleBatchOperation(c *gin.Context) {
	var input struct {
		Operation string   `json:"operation" binding:"required"`
		MediaIDs  []string `json:"media_ids" binding:"required"`
		FolderID  *string  `json:"folder_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	switch input.Operation {
	case "delete":
		if err := database.GetDB().Where("id IN ? AND user_id = ?", input.MediaIDs, userID).Delete(&models.Media{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media"})
			return
		}
	case "move":
		if input.FolderID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Folder ID required for move operation"})
			return
		}
		if err := database.GetDB().Model(&models.Media{}).Where("id IN ? AND user_id = ?", input.MediaIDs, userID).
			Update("folder_id", input.FolderID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move media"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid operation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Batch operation completed",
		"operation":    input.Operation,
		"affected_ids": input.MediaIDs,
	})
}

// BatchTransformMedia handles batch transformation of multiple media files
func BatchTransformMedia(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var operations []BatchOperation
	if err := c.ShouldBindJSON(&operations); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	results := make([]gin.H, 0)
	for _, op := range operations {
		// Find media by ID
		var media models.Media
		if err := database.GetDB().Where("id = ? AND user_id = ?", op.MediaID, userID).
			First(&media).Error; err != nil {
			results = append(results, gin.H{
				"media_id": op.MediaID,
				"error":    "Media not found",
			})
			continue
		}

		// Initialize storage
		storageProvider, err := initializeStorage()
		if err != nil {
			results = append(results, gin.H{
				"media_id": op.MediaID,
				"error":    fmt.Sprintf("Failed to initialize storage: %v", err),
			})
			continue
		}

		// Get internal URL
		internalURL := storageProvider.GetInternalURL(media.Path)

		// Fetch file
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(internalURL)
		if err != nil {
			results = append(results, gin.H{
				"media_id": op.MediaID,
				"error":    fmt.Sprintf("Failed to fetch file: %v", err),
			})
			continue
		}
		defer resp.Body.Close()

		// Check if it's an image
		contentType := media.MimeType
		if !strings.HasPrefix(contentType, "image/") {
			results = append(results, gin.H{
				"media_id": op.MediaID,
				"error":    "Not an image file",
			})
			continue
		}

		// Apply transformations
		transformedImage, err := utils.TransformImage(resp.Body, op.Transformations)
		if err != nil {
			results = append(results, gin.H{
				"media_id": op.MediaID,
				"error":    fmt.Sprintf("Failed to transform image: %v", err),
			})
			continue
		}

		// Generate unique filename for transformed image
		ext := ".jpg"
		if op.Transformations.Format == "png" {
			ext = ".png"
		} else if op.Transformations.Format == "webp" {
			ext = ".webp"
		}
		transformedFilename := fmt.Sprintf("%s_transformed_%d%s",
			strings.TrimSuffix(strings.Split(media.Path, "/")[len(strings.Split(media.Path, "/"))-1], ext),
			time.Now().UnixNano(),
			ext,
		)

		// Upload transformed image
		transformedURL, err := storageProvider.UploadBytes(transformedImage, transformedFilename)
		if err != nil {
			results = append(results, gin.H{
				"media_id": op.MediaID,
				"error":    fmt.Sprintf("Failed to upload transformed image: %v", err),
			})
			continue
		}

		// Create metadata for transformed image
		metadata := map[string]interface{}{
			"original_media_id": media.ID,
			"transformations":   op.Transformations,
		}
		metadataJSON, err := json.Marshal(metadata)
		if err != nil {
			results = append(results, gin.H{
				"media_id": op.MediaID,
				"error":    fmt.Sprintf("Failed to marshal metadata: %v", err),
			})
			continue
		}

		// Create new media record for transformed image
		transformedMedia := models.Media{
			ID:       transformedURL,
			UserID:   userID.(uint),
			FolderID: media.FolderID,
			Filename: transformedFilename,
			Path:     transformedURL,
			MimeType: fmt.Sprintf("image/%s", strings.TrimPrefix(ext, ".")),
			Size:     int64(len(transformedImage)),
			Metadata: metadataJSON,
		}

		if err := database.GetDB().Create(&transformedMedia).Error; err != nil {
			results = append(results, gin.H{
				"media_id": op.MediaID,
				"error":    fmt.Sprintf("Failed to save transformed media: %v", err),
			})
			continue
		}

		results = append(results, gin.H{
			"media_id":             op.MediaID,
			"transformed_id":       transformedMedia.ID,
			"transformed_url":      storageProvider.GetPublicURL(transformedURL),
			"original_filename":    media.Filename,
			"transformed_filename": transformedMedia.Filename,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
	})
}
