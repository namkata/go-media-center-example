package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"go-media-center-example/internal/api/handlers"
	"go-media-center-example/internal/api/middleware"
	"go-media-center-example/internal/config"
	"go-media-center-example/internal/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize Router
	router := gin.Default()

	// Initialize Database
	if err := database.Initialize(cfg); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize Routes
	initRoutes(router)

	// Start Server
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initRoutes(router *gin.Engine) {
	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Auth routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
		}

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.JWTAuth())
		{
			// Media routes
			media := protected.Group("/media")
			{
				media.POST("/upload", handlers.UploadMedia)
				media.GET("/list", handlers.ListMedia)
				media.GET("/:id", handlers.GetMedia)
				media.PUT("/:id", handlers.UpdateMedia)
				media.DELETE("/:id", handlers.DeleteMedia)
				media.POST("/batch", handlers.BatchOperation)
			}

			// Folder routes
			folders := protected.Group("/folders")
			{
				folders.POST("/", handlers.CreateFolder)
				folders.GET("/", handlers.ListFolders)
				folders.PUT("/:id", handlers.UpdateFolder)
				folders.DELETE("/:id", handlers.DeleteFolder)
			}

			// Export routes
			export := protected.Group("/export")
			{
				export.GET("/csv", handlers.ExportCSV)
				export.GET("/json", handlers.ExportJSON)
			}
		}
	}
}
