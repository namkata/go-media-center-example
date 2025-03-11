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
	writer.Write([]string{"ID", "Name", "Type", "Size", "URL", "Created At", "Updated At"})

	// Write data
	for _, m := range media {
		writer.Write([]string{
			fmt.Sprint(m.ID),
			m.Name,
			m.Type,
			fmt.Sprint(m.Size),
			m.URL,
			m.CreatedAt.String(),
			m.UpdatedAt.String(),
		})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate JSON"})
		return
	}

	c.Writer.Write(jsonData)
}
