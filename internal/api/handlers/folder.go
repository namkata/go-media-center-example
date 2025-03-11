package handlers

import (
	"net/http"
	"strconv"

	"go-media-center-example/internal/database"
	"go-media-center-example/internal/models"

	"github.com/gin-gonic/gin"
)

func CreateFolder(c *gin.Context) {
	var input struct {
		Name     string `json:"name" binding:"required,min=1,max=255"`
		ParentID *uint  `json:"parent_id,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: folder name is required"})
		return
	}

	// Validate parent folder if provided
	if input.ParentID != nil {
		// Ensure parent_id is positive
		if *input.ParentID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parent folder ID must be a positive number"})
			return
		}

		var parentFolder models.Folder
		if err := database.GetDB().Where("id = ?", *input.ParentID).First(&parentFolder).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parent folder not found"})
			return
		}
	}

	userID, _ := c.Get("user_id")
	folder := models.Folder{
		Name:     input.Name,
		ParentID: input.ParentID,
		UserID:   userID.(uint),
	}

	if err := database.GetDB().Create(&folder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder"})
		return
	}

	c.JSON(http.StatusCreated, folder)
}

func ListFolders(c *gin.Context) {
	var folders []models.Folder
	userID, _ := c.Get("user_id")
	db := database.GetDB()

	// Parse query parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	parentID := c.Query("parent_id")

	// Base query with user filter
	query := db.Model(&models.Folder{}).Where("user_id = ?", userID)

	// Apply search filter
	if search != "" {
		query = query.Where("name ILIKE ?", "%"+search+"%")
	}

	// Apply parent folder filter
	if parentID != "" {
		if parentID == "root" {
			query = query.Where("parent_id IS NULL")
		} else {
			query = query.Where("parent_id = ?", parentID)
		}
	}

	// Count total before pagination
	var total int64
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count folders"})
		return
	}

	// Apply pagination and fetch results
	offset := (page - 1) * limit
	if err := query.Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&folders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch folders"})
		return
	}

	// Get media count for each folder
	for i := range folders {
		var count int64
		if err := db.Model(&models.Media{}).Where("folder_id = ?", folders[i].ID).Count(&count).Error; err != nil {
			continue
		}
		folders[i].MediaCount = count
	}

	c.JSON(http.StatusOK, gin.H{
		"folders": folders,
		"pagination": gin.H{
			"current_page": page,
			"total_pages":  (total + int64(limit) - 1) / int64(limit),
			"total_items":  total,
			"per_page":     limit,
		},
	})
}

func UpdateFolder(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Name     string `json:"name"`
		ParentID *uint  `json:"parent_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	var folder models.Folder

	if err := database.GetDB().Where("id = ? AND user_id = ?", id, userID).First(&folder).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}

	folder.Name = input.Name
	folder.ParentID = input.ParentID

	if err := database.GetDB().Save(&folder).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update folder"})
		return
	}

	c.JSON(http.StatusOK, folder)
}

func DeleteFolder(c *gin.Context) {
	id := c.Param("id")
	userID, _ := c.Get("user_id")

	// Check if folder has media
	var mediaCount int64
	if err := database.GetDB().Model(&models.Media{}).Where("folder_id = ?", id).Count(&mediaCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check folder contents"})
		return
	}

	if mediaCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete folder containing media"})
		return
	}

	result := database.GetDB().Where("id = ? AND user_id = ?", id, userID).Delete(&models.Folder{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete folder"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Folder deleted successfully"})
}
