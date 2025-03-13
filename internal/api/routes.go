package api

import (
	"go-media-center-example/internal/api/handlers"
	"go-media-center-example/internal/api/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRoutes configures all application routes
func SetupRoutes(router *gin.Engine) {
	// API v1 group
	v1 := router.Group("/api/v1")
	{
		// Public routes
		setupPublicRoutes(v1)

		// Protected routes
		protected := v1.Group("/")
		protected.Use(middleware.JWTAuth())
		setupProtectedRoutes(protected)
	}
}

// setupPublicRoutes configures public routes that don't require authentication
func setupPublicRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", handlers.Register)
		auth.POST("/login", handlers.Login)
	}

	// Serve media files (if public access is needed)
	media := rg.Group("/media/files")
	{
		media.GET("/:filename", handlers.ServeMediaFile)
	}
}

// setupProtectedRoutes configures routes that require authentication
func setupProtectedRoutes(rg *gin.RouterGroup) {
	// Media routes
	media := rg.Group("/media")
	{
		media.POST("/upload", handlers.UploadMedia)
		media.POST("/url", handlers.UploadMediaFromURL)
		media.POST("/batch", handlers.BulkUploadMedia)
		media.GET("/list", handlers.ListMedia)
		media.PUT("/:id", handlers.UpdateMedia)
		media.GET("/:id", handlers.GetMedia)
		media.DELETE("/:id", handlers.DeleteMedia)

		// Transform API Examples:
		// 1. Basic resize:
		//    POST /api/v1/media/{id}/transform?width=800&height=600
		//
		// 2. Resize with specific fit mode:
		//    POST /api/v1/media/{id}/transform?width=800&height=600&fit=cover
		//    Fit options: contain, cover, fill
		//
		// 3. Format conversion with quality:
		//    POST /api/v1/media/{id}/transform?format=webp&quality=80
		//    Formats: jpeg, png, webp
		//    Quality: 1-100
		//
		// 4. Using presets:
		//    POST /api/v1/media/{id}/transform?preset=thumbnail
		//    Available presets:
		//    - thumbnail: 150x150 cover
		//    - social: 1200x630 contain
		//    - avatar: 300x300 cover
		//    - banner: 1920x400 cover
		//
		// 5. Crop operation:
		//    POST /api/v1/media/{id}/transform?crop=100,100,500,300
		//    Format: x,y,width,height
		//
		// 6. Combined operations:
		//    POST /api/v1/media/{id}/transform?width=800&height=600&format=webp&quality=80&fit=cover
		//
		// 7. Force fresh transformation (skip cache):
		//    Add fresh=true to any transform request
		//    Example: /api/v1/media/{id}/transform?width=800&fresh=true
		media.POST("/:id/transform", handlers.TransformMedia)
	}

	// Folder routes
	folders := rg.Group("/folders")
	{
		folders.POST("/", handlers.CreateFolder)
		folders.GET("/", handlers.ListFolders)
		folders.PUT("/:id", handlers.UpdateFolder)
		folders.DELETE("/:id", handlers.DeleteFolder)
	}

	// Export routes
	export := rg.Group("/export")
	{
		export.GET("/csv", handlers.ExportCSV)
		export.GET("/json", handlers.ExportJSON)
	}
}
