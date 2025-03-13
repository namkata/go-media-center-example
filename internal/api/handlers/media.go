package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	"gorm.io/gorm"
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
			"public_url":        cfg.Storage.S3.PublicURL,
		}
	}

	return storage.NewStorage(provider, storageConfig)
}

// ServeMediaFile handles serving media files through the application server
func ServeMediaFile(c *gin.Context) {
	filename := c.Param("filename")
	userID, _ := c.Get("user_id")

	// Parse transformation options
	queryParams := make(map[string]string)
	for k := range c.Request.URL.Query() {
		queryParams[k] = c.Query(k)
	}

	transformOptions := utils.TransformationOptions{
		Width:   utils.ParseIntOption(queryParams["width"]),
		Height:  utils.ParseIntOption(queryParams["height"]),
		Fit:     queryParams["fit"],
		Crop:    queryParams["crop"],
		Quality: utils.ParseIntOption(queryParams["quality"]),
		Format:  queryParams["format"],
		Preset:  queryParams["preset"],
		Fresh:   queryParams["fresh"] == "true",
	}

	// Find media by filename
	var media models.Media
	if err := database.GetDB().Where("path LIKE ?", "%"+filename+"%").
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
	internalURL := storageProvider.GetInternalURL(media.Path)

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
	contentType := media.MimeType

	// Check if it's an image that needs transformation
	if strings.HasPrefix(contentType, "image/") && !transformOptions.IsEmpty() {
		// Apply transformations
		transformedImage, err := utils.TransformImage(resp.Body, transformOptions)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to transform image: %v", err)})
			return
		}

		// Set appropriate content type based on format
		switch transformOptions.Format {
		case "png":
			contentType = "image/png"
		case "webp":
			contentType = "image/webp"
		default:
			contentType = "image/jpeg"
		}

		// Set cache control headers
		if !transformOptions.Fresh {
			c.Header("Cache-Control", "public, max-age=31536000") // Cache for 1 year
			c.Header("ETag", fmt.Sprintf("%s-%v", filename, transformOptions))
		} else {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		}

		// Set content type and filename
		c.Header("Content-Type", contentType)
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", media.Filename))

		// Write the transformed image
		c.Data(http.StatusOK, contentType, transformedImage)
		return
	}

	// For non-image files or no transformation needed
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", media.Filename))

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

	// Open the file for reading
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to open file: %v", err)})
		return
	}
	defer f.Close()

	// Upload file to storage
	fileID, err := storageProvider.Upload(f, file.Filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to upload file: %v", err)})
		return
	}

	// Get both internal and public URLs for the file
	fileInternalURL := storageProvider.GetInternalURL(fileID)
	filePublicURL := storageProvider.GetPublicURL(fileID)

	// Get folder ID if provided
	folderID := c.PostForm("folder_id")
	var fID *string
	if folderID != "" {
		fID = &folderID
		// Verify folder exists and belongs to user
		var folder models.Folder
		if err := database.GetDB().Where("id = ? AND user_id = ?", folderID, userID).First(&folder).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder ID"})
			return
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
	metadata := map[string]interface{}{
		"original_name": file.Filename,
		"file_id":       fileID,
		"internal_url":  fileInternalURL,
		"public_url":    filePublicURL,
		"technical":     mediaMetadata,
	}

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to marshal metadata: %v", err)})
		return
	}

	// Save to database
	media := models.Media{
		ID:       fileID,
		UserID:   userID.(uint),
		FolderID: fID,
		Filename: file.Filename,
		Path:     fileID,
		MimeType: mediaMetadata.MimeType,
		Size:     file.Size,
		Metadata: metadataJSON,
	}

	// Create with transaction
	tx := database.GetDB().Begin()
	if err := tx.Model(&models.Media{}).Create(&media).Error; err != nil {
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
	return storageProvider.GetPublicURL(mediaItem.Path), nil
}

func getFileInternalURL(mediaItem *models.Media) (string, error) {
	storageProvider, err := initializeStorage()
	if err != nil {
		return "", err
	}
	return storageProvider.GetInternalURL(mediaItem.Path), nil
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
		query = query.Where("media.mime_type LIKE ?", fileType+"%")
	}

	if search != "" {
		query = query.Where("media.filename ILIKE ?", "%"+search+"%")
	}

	if folderID != "" {
		query = query.Where("media.folder_id = ?", folderID)
	}

	// Filter by tags if provided
	if len(tags) > 0 {
		query = query.Joins("LEFT JOIN media_tags ON media_tags.media_id = media.id").
			Joins("LEFT JOIN tags ON tags.id = media_tags.tag_id").
			Where("tags.name IN ?", tags).
			Group("media.id").
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
	offset := (page - 1) * limit
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
		// Parse existing metadata
		var metadata map[string]interface{}
		if len(media[i].Metadata) > 0 {
			if err := json.Unmarshal(media[i].Metadata, &metadata); err != nil {
				metadata = make(map[string]interface{})
			}
		} else {
			metadata = make(map[string]interface{})
		}

		// Add URLs to metadata
		if fileURL, err := getFileURL(&media[i]); err == nil {
			metadata["public_url"] = fileURL
		}
		if internalURL, err := getFileInternalURL(&media[i]); err == nil {
			metadata["internal_url"] = internalURL
		}

		// Convert back to JSON
		if metadataJSON, err := json.Marshal(metadata); err == nil {
			media[i].Metadata = metadataJSON
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
	presignedURL, err := storageProvider.GetPresignedURL(media.Path, time.Duration(expiration)*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to generate presigned URL: %v", err)})
		return
	}

	// Add URLs to metadata
	var metadata map[string]interface{}
	if len(media.Metadata) > 0 {
		if err := json.Unmarshal(media.Metadata, &metadata); err != nil {
			metadata = make(map[string]interface{})
		}
	} else {
		metadata = make(map[string]interface{})
	}

	// Add presigned URL to metadata
	metadata["presigned_url"] = presignedURL
	metadata["url_expiration"] = expiration

	// Convert back to JSON
	if metadataJSON, err := json.Marshal(metadata); err == nil {
		media.Metadata = metadataJSON
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
		Filename string   `json:"filename"`
		FolderID *string  `json:"folder_id"`
		Metadata []byte   `json:"metadata"`
		Tags     []string `json:"tags"`
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
		"filename":  input.Filename,
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
	if err := storageProvider.Delete(media.Path); err != nil {
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

// TransformMedia handles image transformation requests
func TransformMedia(c *gin.Context) {
	mediaID := c.Param("id")
	if mediaID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Media ID is required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Get media from database
	media, err := models.GetMediaByID(mediaID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Media not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve media"})
		return
	}

	// Check if media belongs to user
	if media.UserID != userID.(uint) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Check if media is an image
	if !strings.HasPrefix(media.MimeType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Media is not an image"})
		return
	}

	// Parse transformation options from query parameters
	options := utils.TransformationOptions{
		Width:   utils.ParseIntOption(c.Query("width")),
		Height:  utils.ParseIntOption(c.Query("height")),
		Fit:     c.Query("fit"),
		Crop:    c.Query("crop"),
		Quality: utils.ParseIntOption(c.Query("quality")),
		Format:  c.Query("format"),
		Preset:  c.Query("preset"),
		Fresh:   c.Query("fresh") == "true",
	}

	// Log transformation options for debugging
	fmt.Printf("Transformation options: %+v\n", options)

	// Validate transformation options
	if err := options.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid transformation parameters",
			"details": err.Error(),
		})
		return
	}

	// Apply preset if specified
	if options.Preset != "" {
		if err := utils.ApplyPreset(&options, options.Preset); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid preset",
				"details": err.Error(),
			})
			return
		}
	}

	// Get storage provider
	storageProvider := storage.GetProvider()
	if storageProvider == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage provider not initialized"})
		return
	}

	// Read original file
	reader, err := storageProvider.Download(media.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to read original file",
			"details": err.Error(),
		})
		return
	}
	defer reader.Close()

	// Generate cache key for transformed image
	cacheKey := fmt.Sprintf(
		"%s_w%d_h%d_f%s_c%s_q%d_%s",
		media.ID,
		options.Width,
		options.Height,
		options.Fit,
		options.Crop,
		options.Quality,
		options.Format,
	)

	// Check if transformed version exists
	if !options.Fresh {
		if cachedReader, err := storageProvider.Download(cacheKey); err == nil {
			defer cachedReader.Close()
			// Read the entire file into memory since we can't seek on the reader
			data, err := io.ReadAll(cachedReader)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read cached file"})
				return
			}
			c.Header("X-Cache", "HIT")
			c.Data(http.StatusOK, media.MimeType, data)
			return
		}
	}

	// Transform image
	transformed, err := utils.TransformImage(reader, options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to transform image",
			"details": err.Error(),
		})
		return
	}

	// Upload transformed version
	if _, err := storageProvider.UploadBytes(transformed, cacheKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transformed image"})
		return
	}

	// Set cache control headers
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Header("X-Cache", "MISS")

	// Set appropriate content type based on format
	contentType := media.MimeType
	if options.Format != "" {
		switch options.Format {
		case "png":
			contentType = "image/png"
		case "webp":
			contentType = "image/webp"
		default:
			contentType = "image/jpeg"
		}
	}

	// Serve transformed image
	c.Data(http.StatusOK, contentType, transformed)
}
