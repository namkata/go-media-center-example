package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-media-center-example/internal/config"
	"go-media-center-example/internal/database"
	"go-media-center-example/internal/models"
	"go-media-center-example/internal/storage"
	"go-media-center-example/internal/utils"

	"github.com/gin-gonic/gin"
)

// initializeStorage creates a new storage provider based on configuration
const (
    defaultURLExpiration = 24 * time.Hour // Default URL expiration time
)

func initializeStorage() (storage.Storage, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	var provider storage.StorageProvider
	switch strings.ToLower(cfg.Storage.Provider) {
	case "seaweedfs":
		provider = storage.SeaweedFS
	case "s3":
		provider = storage.S3
	default:
		return nil, fmt.Errorf("unsupported storage provider: %s", cfg.Storage.Provider)
	}

	storageConfig := make(map[string]string)

	switch provider {
	case storage.SeaweedFS:
		storageConfig = map[string]string{
			"master_url":   cfg.Storage.SeaweedFS.MasterURL,
			"internal_url": fmt.Sprintf("http://localhost:%d", cfg.Storage.SeaweedFS.VolumePort),
			"public_url":   fmt.Sprintf("http://localhost:%s", cfg.Server.Port),
		}
	case storage.S3:
		storageConfig = map[string]string{
			"region":            cfg.Storage.S3.Region,
			"access_key_id":     cfg.Storage.S3.AccessKeyID,
			"secret_access_key": cfg.Storage.S3.SecretAccessKey,
			"bucket":            cfg.Storage.S3.BucketName,
			"endpoint":          cfg.Storage.S3.Endpoint,
			"force_path_style":  "true",
			"url_expiration":    defaultURLExpiration.String(),
		}

		// Set public URL if provided, otherwise construct it
		if cfg.Storage.S3.PublicURL != "" {
			storageConfig["public_url"] = cfg.Storage.S3.PublicURL
		} else if cfg.Storage.S3.Endpoint != "" {
			storageConfig["public_url"] = fmt.Sprintf("%s/%s", cfg.Storage.S3.Endpoint, cfg.Storage.S3.BucketName)
		}
	}

	return storage.NewStorage(provider, storageConfig)
}

// ServeMediaFile handles serving media files through the application server
func ServeMediaFile(c *gin.Context) {
	filename := c.Param("filename")
	userID, _ := c.Get("user_id")

	// Parse width and height parameters
	width := c.Query("w")
	height := c.Query("h")
	fit := c.DefaultQuery("fit", "contain") // contain, cover, fill

	// Find media by filename
	var media models.Media
	if err := database.GetDB().Where("url LIKE ?", "%"+filename+"%").
		Where("user_id = ?", userID).
		First(&media).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
		return
	}

	// Initialize storage
	storageProvider, err := initializeStorage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to initialize storage: %v", err)})
		return
	}

	// Get internal URL for the file using the stored file ID
	internalURL := storageProvider.GetInternalURL(media.URL)

	// Create HTTP client with appropriate timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Fetch file from storage using internal URL
	resp, err := client.Get(internalURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to fetch file: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Get content type
	contentType := ""
	if ct, ok := media.Metadata["mime_type"].(string); ok {
		contentType = ct
	} else {
		contentType = resp.Header.Get("Content-Type")
	}

	// Check if it's an image that needs resizing
	if (width != "" || height != "") && strings.HasPrefix(contentType, "image/") {
		// Parse dimensions
		var w, h int
		if width != "" {
			w, _ = strconv.Atoi(width)
		}
		if height != "" {
			h, _ = strconv.Atoi(height)
		}

		// Process image if valid dimensions
		if w > 0 || h > 0 {
			resizedImage, err := utils.ProcessImageWithSize(resp.Body, w, h, fit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to process image: %v", err)})
				return
			}

			// Set headers
			c.Header("Content-Type", contentType)
			if originalName, ok := media.Metadata["original_name"].(string); ok {
				c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", originalName))
			}

			// Write the processed image
			c.Writer.Write(resizedImage)
			return
		}
	}

	// Set content type
	c.Header("Content-Type", contentType)

	// Set filename for download
	if originalName, ok := media.Metadata["original_name"].(string); ok {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", originalName))
	}

	// Stream the original file
	c.DataFromReader(http.StatusOK, resp.ContentLength, contentType, resp.Body, nil)
}

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

	// Extract detailed metadata
	mediaMetadata, err := utils.ExtractMetadata(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to extract metadata: %v", err)})
		return
	}

	// Initialize storage
	storageProvider, err := initializeStorage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to initialize storage: %v", err)})
		return
	}

	// Upload file to storage
	fileID, err := storageProvider.Upload(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file: %v", err)})
		return
	}

	// Get both internal and public URLs for the file
	fileInternalURL := storageProvider.GetInternalURL(fileID)
	filePublicURL := storageProvider.GetURL(fileID)

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

	// Create metadata combining file info and technical metadata
	metadata := models.JSON{
		"original_name": file.Filename,
		"file_id":       fileID,
		"internal_url":  fileInternalURL,
		"public_url":    filePublicURL,
		"technical":     mediaMetadata,
	}

	// Save to database
	media := models.Media{
		Name:     file.Filename,
		Type:     mediaMetadata.FileType,
		Size:     file.Size,
		URL:      fileID,
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
		storageProvider.Delete(fileID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to save media metadata: %v", err)})
		return
	}
	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "File uploaded successfully",
		"media":   media,
	})
}

// Add helper methods to get file URLs
func getFileURL(mediaItem *models.Media) (string, error) {
	storageProvider, err := initializeStorage()
	if err != nil {
		return "", err
	}
	return storageProvider.GetURL(mediaItem.URL), nil
}

func getFileInternalURL(mediaItem *models.Media) (string, error) {
	storageProvider, err := initializeStorage()
	if err != nil {
		return "", err
	}
	return storageProvider.GetInternalURL(mediaItem.URL), nil
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

	// Add file URLs to the response
	for i := range media {
		if fileURL, err := getFileURL(&media[i]); err == nil {
			if media[i].Metadata == nil {
				media[i].Metadata = models.JSON{}
			}
			media[i].Metadata["public_url"] = fileURL
		}
		if internalURL, err := getFileInternalURL(&media[i]); err == nil {
			if media[i].Metadata == nil {
				media[i].Metadata = models.JSON{}
			}
			media[i].Metadata["internal_url"] = internalURL
		}
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

    // Get expiration from query parameter, default to 24 hours
    expirationStr := c.DefaultQuery("expires", "86400") // 24 hours in seconds
    expiration, err := strconv.Atoi(expirationStr)
    if err != nil {
        expiration = int(defaultURLExpiration.Seconds())
    }

    var media models.Media
    if err := database.GetDB().
        Preload("Tags").
        Where("id = ? AND user_id = ?", id, userID).
        First(&media).Error; err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Media not found: %v", err)})
        return
    }

    // Initialize storage for presigned URL
    storageProvider, err := initializeStorage()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to initialize storage: %v", err)})
        return
    }

    // Generate presigned URL
    presignedURL, err := storageProvider.GetPresignedURL(media.URL, time.Duration(expiration)*time.Second)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate presigned URL: %v", err)})
        return
    }

    // Add URLs to metadata
    if media.Metadata == nil {
        media.Metadata = models.JSON{}
    }
    media.Metadata["presigned_url"] = presignedURL
    media.Metadata["expires_in"] = expiration

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

	// Initialize storage
	storageProvider, err := initializeStorage()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to initialize storage: %v", err)})
		return
	}

	// Delete file from storage
	if err := storageProvider.Delete(media.URL); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to delete file: %v", err)})
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
