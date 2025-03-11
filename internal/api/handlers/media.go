package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go-media-center-example/internal/config"
	"go-media-center-example/internal/database"
	"go-media-center-example/internal/models"
	"go-media-center-example/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func UploadMedia(c *gin.Context) {
	cfg, _ := config.Load()
	userID, _ := c.Get("user_id")

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}

	if file.Size > cfg.Storage.MaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large"})
		return
	}

	// Generate unique filename
	filename := uuid.New().String() + filepath.Ext(file.Filename)

	// Read file
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}
	defer f.Close()

	fileBytes := make([]byte, file.Size)
	if _, err := f.Read(fileBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	// Save file
	filePath, err := utils.SaveFile(fileBytes, filename, cfg.Storage.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	// Process image if applicable
	fileType := utils.GetFileType(filename)
	if fileType == "image" {
		if err := utils.ProcessImage(filePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process image"})
			return
		}
	}

	// Get folder ID if provided
	folderID := c.PostForm("folder_id")
	var fID *uint
	if folderID != "" {
		if id, err := strconv.ParseUint(folderID, 10, 32); err == nil {
			converted := uint(id)
			// Verify folder exists and belongs to user
			var folder models.Folder
			if err := database.GetDB().Where("id = ? AND user_id = ?", converted, userID).First(&folder).Error; err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
				return
			}
			fID = &converted
		}
	}

	// Handle tags if provided
	var tags []models.Tag
	if tagNames := c.PostFormArray("tags"); len(tagNames) > 0 {
		for _, name := range tagNames {
			var tag models.Tag
			// Find or create tag
			result := database.GetDB().Where("name = ?", name).FirstOrCreate(&tag, models.Tag{Name: name})
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process tags"})
				return
			}
			tags = append(tags, tag)
		}
	}

	// Create metadata with additional info
	metadata := models.JSON{
		"original_name": file.Filename,
		"mime_type":     file.Header.Get("Content-Type"),
		"uploaded_at":   time.Now().Format(time.RFC3339),
	}

	// Save to database
	media := models.Media{
		Name:     file.Filename,
		Type:     fileType,
		Size:     file.Size,
		URL:      "/storage/" + filename,
		UserID:   userID.(uint),
		FolderID: fID,
		Metadata: metadata,
		Tags:     tags,
	}

	// Create with transaction
	tx := database.GetDB().Begin()
	if err := tx.Create(&media).Error; err != nil {
		tx.Rollback()
		// Clean up uploaded file
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save media metadata: %v", err)})
		return
	}
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"media":   media,
	})
}

func ListMedia(c *gin.Context) {
	var media []models.Media
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	fileType := c.Query("type")
	search := c.Query("search")
	folderID := c.Query("folder_id")
	tags := c.QueryArray("tags")

	// Base query with user filter
	query := db.Table("media").Select("DISTINCT media.*").Where("media.user_id = ?", userID)

	// Apply filters
	if fileType != "" {
		query = query.Where("media.type = ?", fileType)
	}

	if search != "" {
		query = query.Where("media.name ILIKE ?", "%"+search+"%")
	}

	if folderID != "" {
		query = query.Where("media.folder_id = ?", folderID)
	}

	// Filter by tags if provided
	if len(tags) > 0 {
		query = query.Joins("LEFT JOIN media_tags ON media_tags.media_id = media.id").
			Joins("LEFT JOIN tags ON tags.id = media_tags.tag_id").
			Where("tags.name IN ?", tags).
			Group("media.id, media.name, media.type, media.size, media.url, "+
				"media.folder_id, media.user_id, media.metadata, "+
				"media.created_at, media.updated_at, media.deleted_at").
			Having("COUNT(DISTINCT tags.name) = ?", len(tags))
	}

	// Count total before pagination
	var total int64
	countQuery := db.Table("(?) as counted_media", query).Count(&total)
	if countQuery.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to count media: %v", countQuery.Error)})
		return
	}

	// Apply pagination and fetch results
	offset := 10
	if err := query.Offset(offset).Limit(limit).
		Order("media.created_at DESC").
		Scan(&media).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch media: %v", err)})
		return
	}

	// Load tags separately to avoid JSON scanning issues
	if err := db.Preload("Tags").Find(&media).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to load tags: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"media": media,
		"pagination": gin.H{
			"current_page": page,
			"total_pages":  (total + int64(limit) - 1) / int64(limit),
			"total_items":  total,
			"per_page":     limit,
		},
	})
}

func GetMedia(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	var media models.Media
	if err := database.GetDB().
		Preload("Tags").
		Where("id = ? AND user_id = ?", id, userID).
		First(&media).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Media not found: %v", err)})
		return
	}

	// Get folder info if media is in a folder
	if media.FolderID != nil {
		var folder models.Folder
		if err := database.GetDB().Select("id, name").First(&folder, media.FolderID).Error; err == nil {
			c.JSON(http.StatusOK, gin.H{
				"media": media,
				"folder": gin.H{
					"id":   folder.ID,
					"name": folder.Name,
				},
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"media": media})
}

func UpdateMedia(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	var input struct {
		Name     string      `json:"name"`
		FolderID *uint       `json:"folder_id"`
		Metadata models.JSON `json:"metadata"`
		Tags     []string    `json:"tags"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var media models.Media
	if err := database.GetDB().Where("id = ? AND user_id = ?", id, userID).First(&media).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	updates := map[string]interface{}{
		"name":      input.Name,
		"folder_id": input.FolderID,
		"metadata":  input.Metadata,
	}

	if err := database.GetDB().Model(&media).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update media"})
		return
	}

	c.JSON(http.StatusOK, media)
}

func DeleteMedia(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	var media models.Media
	if err := database.GetDB().Where("id = ? AND user_id = ?", id, userID).First(&media).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	// Delete file
	cfg, _ := config.Load()
	filePath := filepath.Join(cfg.Storage.Path, filepath.Base(media.URL))
	if err := os.Remove(filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	// Delete from database
	if err := database.GetDB().Delete(&media).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Media deleted successfully"})
}

func BatchOperation(c *gin.Context) {
	var input struct {
		Operation string   `json:"operation" binding:"required"`
		MediaIDs  []string `json:"media_ids" binding:"required"`
		FolderID  *uint    `json:"folder_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	db := database.GetDB()

	switch input.Operation {
	case "delete":
		if err := db.Where("id IN ? AND user_id = ?", input.MediaIDs, userID).Delete(&models.Media{}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media"})
			return
		}
	case "move":
		if input.FolderID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Folder ID required for move operation"})
			return
		}
		if err := db.Model(&models.Media{}).Where("id IN ? AND user_id = ?", input.MediaIDs, userID).
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
