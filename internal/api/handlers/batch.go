package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"go-media-center-example/internal/config"
	"go-media-center-example/internal/database"
	"go-media-center-example/internal/models"
	"go-media-center-example/internal/storage"
	"go-media-center-example/internal/utils"
)

// BatchOperation represents a batch operation request
type BatchOperation struct {
	MediaID         uint                        `json:"media_id"`
	Transformations utils.TransformationOptions `json:"transformations"`
}

// URLUploadRequest represents a URL to upload
type URLUploadRequest struct {
	URL      string   `json:"url" binding:"required"`
	Filename string   `json:"filename"`
	Tags     []string `json:"tags"`
}

// BulkURLUpload handles uploading multiple files from URLs
func BulkURLUpload(c *gin.Context) {
	cfg, _ := config.Load()
	userID, _ := c.Get("user_id")

	var input struct {
		URLs     []URLUploadRequest `json:"urls" binding:"required"`
		FolderID string             `json:"folder_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid request: %v", err)})
		return
	}

	if len(input.URLs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No URLs provided"})
		return
	}

	// Verify folder if provided
	var fID *string
	if input.FolderID != "" {
		fID = &input.FolderID
		var folder models.Folder
		if err := database.GetDB().Where("id = ? AND user_id = ?", input.FolderID, userID).First(&folder).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
			return
		}
	}

	// Initialize storage
	storageProvider, err := initializeStorage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to initialize storage: %v", err)})
		return
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 60 * time.Second, // Longer timeout for potentially large files
	}

	// Process URLs concurrently with a limit
	maxConcurrent := 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	results := make([]gin.H, len(input.URLs))
	for i, urlReq := range input.URLs {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(i int, urlReq URLUploadRequest) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			result := processURLUpload(client, storageProvider, urlReq, fID, userID.(uint), cfg.Storage.MaxUploadSize)
			results[i] = result
		}(i, urlReq)
	}

	wg.Wait()

	// Count successful uploads
	successCount := 0
	for _, result := range results {
		if result["success"].(bool) {
			successCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Bulk URL upload completed",
		"total":         len(input.URLs),
		"success_count": successCount,
		"results":       results,
	})
}

// processURLUpload handles a single URL upload
func processURLUpload(client *http.Client, storageProvider storage.Storage, urlReq URLUploadRequest, folderID *string, userID uint, maxUploadSize int64) gin.H {
	// Download file from URL
	resp, err := client.Get(urlReq.URL)
	if err != nil {
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   fmt.Sprintf("Failed to download: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   fmt.Sprintf("Failed to download: status code %d", resp.StatusCode),
		}
	}

	// Check content length if available
	if resp.ContentLength > 0 && resp.ContentLength > maxUploadSize {
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   "File too large",
		}
	}

	// Determine filename if not provided
	filename := urlReq.Filename
	if filename == "" {
		// Try to get filename from URL
		urlPath := resp.Request.URL.Path
		filename = filepath.Base(urlPath)
		if filename == "" || filename == "." || filename == "/" {
			// Generate a timestamp-based filename with extension from content type
			ext := ".bin"
			contentType := resp.Header.Get("Content-Type")
			if strings.HasPrefix(contentType, "image/") {
				switch contentType {
				case "image/jpeg":
					ext = ".jpg"
				case "image/png":
					ext = ".png"
				case "image/gif":
					ext = ".gif"
				case "image/webp":
					ext = ".webp"
				}
			} else if strings.HasPrefix(contentType, "video/") {
				switch contentType {
				case "video/mp4":
					ext = ".mp4"
				case "video/quicktime":
					ext = ".mov"
				case "video/x-msvideo":
					ext = ".avi"
				}
			}
			filename = fmt.Sprintf("download_%d%s", time.Now().Unix(), ext)
		}
	}

	// Upload file to storage
	fileID, err := storageProvider.Upload(resp.Body, filename)
	if err != nil {
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   fmt.Sprintf("Failed to upload file: %v", err),
		}
	}

	// Get file size and metadata
	// We need to download the file again to get metadata
	fileResp, err := client.Get(storageProvider.GetInternalURL(fileID))
	if err != nil {
		// Clean up the uploaded file if we can't get metadata
		storageProvider.Delete(fileID)
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   fmt.Sprintf("Failed to process file: %v", err),
		}
	}
	defer fileResp.Body.Close()

	// Create a temporary file to extract metadata
	tempFile, err := os.CreateTemp("", "url-download-*")
	if err != nil {
		storageProvider.Delete(fileID)
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   fmt.Sprintf("Failed to process file: %v", err),
		}
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Copy the file content to the temp file
	fileSize, err := io.Copy(tempFile, fileResp.Body)
	if err != nil {
		storageProvider.Delete(fileID)
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   fmt.Sprintf("Failed to process file: %v", err),
		}
	}

	// Check file size again
	if fileSize > maxUploadSize {
		storageProvider.Delete(fileID)
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   "File too large",
		}
	}

	// Rewind the temp file
	tempFile.Seek(0, 0)

	// Read the first 512 bytes to detect content type
	buffer := make([]byte, 512)
	_, err = tempFile.Read(buffer)
	if err != nil && err != io.EOF {
		storageProvider.Delete(fileID)
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   fmt.Sprintf("Failed to process file: %v", err),
		}
	}

	// Reset file pointer
	tempFile.Seek(0, 0)

	// Detect content type
	contentType := http.DetectContentType(buffer)

	// Create basic metadata
	mediaMetadata := &utils.MediaMetadata{
		FileType:   utils.GetFileType(filename),
		MimeType:   contentType,
		Size:       fileSize,
		UploadedAt: time.Now().Format(time.RFC3339),
		Format:     strings.TrimPrefix(filepath.Ext(filename), "."),
	}

	// Get both internal and public URLs for the file
	fileInternalURL := storageProvider.GetInternalURL(fileID)
	filePublicURL := storageProvider.GetPublicURL(fileID)

	// Handle tags if provided
	var tags []models.Tag
	if len(urlReq.Tags) > 0 {
		for _, name := range urlReq.Tags {
			var tag models.Tag
			// Find or create tag
			result := database.GetDB().Where("name = ?", name).FirstOrCreate(&tag, models.Tag{Name: name})
			if result.Error != nil {
				storageProvider.Delete(fileID)
				return gin.H{
					"url":     urlReq.URL,
					"success": false,
					"error":   "Failed to process tags",
				}
			}
			tags = append(tags, tag)
		}
	}

	// Create metadata combining file info and technical metadata
	metadata := map[string]interface{}{
		"original_name": filename,
		"source_url":    urlReq.URL,
		"file_id":       fileID,
		"internal_url":  fileInternalURL,
		"public_url":    filePublicURL,
		"technical":     mediaMetadata,
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		storageProvider.Delete(fileID)
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   fmt.Sprintf("Failed to marshal metadata: %v", err),
		}
	}

	// Save to database
	media := models.Media{
		ID:       fileID,
		UserID:   userID,
		FolderID: folderID,
		Filename: filename,
		Path:     fileID,
		MimeType: mediaMetadata.MimeType,
		Size:     fileSize,
		Metadata: metadataJSON,
	}

	// Create with transaction
	tx := database.GetDB().Begin()
	if err := tx.Model(&models.Media{}).Create(&media).Error; err != nil {
		tx.Rollback()
		// Clean up uploaded file
		storageProvider.Delete(fileID)
		return gin.H{
			"url":     urlReq.URL,
			"success": false,
			"error":   fmt.Sprintf("Failed to save media metadata: %v", err),
		}
	}

	// Associate tags if any
	if len(tags) > 0 {
		if err := tx.Model(&media).Association("Tags").Append(&tags); err != nil {
			tx.Rollback()
			storageProvider.Delete(fileID)
			return gin.H{
				"url":     urlReq.URL,
				"success": false,
				"error":   "Failed to associate tags",
			}
		}
	}

	tx.Commit()

	return gin.H{
		"url":      urlReq.URL,
		"success":  true,
		"media_id": media.ID,
		"filename": filename,
	}
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
