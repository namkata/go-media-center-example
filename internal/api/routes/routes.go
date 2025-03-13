package routes

import (
	"go-media-center-example/internal/api/handlers"
	"go-media-center-example/internal/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all the routes for the application
func SetupRoutes(router *gin.Engine) {
	// Public routes
	public := router.Group("/")
	{
		public.GET("/health", handlers.HealthCheck)
		public.POST("/auth/register", handlers.Register)
		public.POST("/auth/login", handlers.Login)
	}

	// API routes
	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())

	// Media routes
	media := api.Group("/media")
	{
		media.POST("", handlers.UploadMedia)
		media.POST("/url", handlers.UploadMediaFromURL)
		media.POST("/bulk", handlers.BulkUploadMedia)
		media.GET("", handlers.ListMedia)
		media.GET("/:id", handlers.GetMedia)
		media.GET("/:id/transform", handlers.TransformMedia)
		media.DELETE("/:id", handlers.DeleteMedia)
		media.GET("/serve/:id", handlers.ServeMediaFile)
		media.POST("/batch", handlers.HandleBatchOperation)
		media.POST("/batch/transform", handlers.BatchTransformMedia)
	}

	// Folder routes
	folders := api.Group("/folders")
	{
		folders.POST("", handlers.CreateFolder)
		folders.GET("", handlers.ListFolders)
		folders.GET("/:id", handlers.GetFolder)
		folders.PUT("/:id", handlers.UpdateFolder)
		folders.DELETE("/:id", handlers.DeleteFolder)
	}
}
