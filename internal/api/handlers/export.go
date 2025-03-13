package handlers

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"

	"go-media-center-example/internal/models"

	"github.com/gin-gonic/gin"

	"go-media-center-example/internal/database"
)

func ExportCSV(c *gin.Context) {
	var media []models.Media
	userID, _ := c.Get("user_id")

	if err := database.GetDB().Where("user_id = ?", userID).Find(&media).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
		return
	}

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment;filename=media_export.csv")

	writer := csv.NewWriter(c.Writer)
	// Write header
	if err := writer.Write([]string{"ID", "Filename", "MimeType", "Size", "Path", "Created At", "Updated At"}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV header"})
		return
	}

	// Write data
	for _, m := range media {
		if err := writer.Write([]string{
			m.ID,
			m.Filename,
			m.MimeType,
			fmt.Sprint(m.Size),
			m.Path,
			m.CreatedAt.String(),
			m.UpdatedAt.String(),
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write CSV data"})
			return
		}
	}

	writer.Flush()
}

func ExportJSON(c *gin.Context) {
	var media []models.Media
	userID, _ := c.Get("user_id")

	if err := database.GetDB().Where("user_id = ?", userID).Find(&media).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch media"})
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment;filename=media_export.json")

	jsonData, err := json.MarshalIndent(media, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal JSON"})
		return
	}

	c.Data(http.StatusOK, "application/json", jsonData)
}
