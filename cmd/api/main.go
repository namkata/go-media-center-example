package main

import (
	"log"

	"github.com/gin-gonic/gin"

	_ "go-media-center-example/docs" // Import swagger docs
	"go-media-center-example/internal/api"
	"go-media-center-example/internal/config"
	"go-media-center-example/internal/database"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title           Media Center API
// @version         1.0
// @description     A media management system with support for images, videos, and documents
// @termsOfService  http://example.com/terms/

// @contact.name   API Support
// @contact.url    http://example.com/support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load configuration:", err)
	}

	// Initialize Router
	router := gin.Default()

	// Configure trusted proxies
	// For development, if you're behind a reverse proxy (like nginx), you might want to trust local networks
	if cfg.Server.Env == "development" {
		// Trust local networks and common proxy addresses
		router.SetTrustedProxies([]string{
			"127.0.0.1",      // localhost
			"::1",            // localhost IPv6
			"10.0.0.0/8",     // private network
			"172.16.0.0/12",  // private network
			"192.168.0.0/16", // private network
		})
	} else {
		// For production, you should explicitly set your trusted proxy IPs
		// Example: router.SetTrustedProxies([]string{"192.168.1.2"})
		// Or if you don't have any proxy:
		router.SetTrustedProxies(nil)
	}

	// Initialize Database
	if err := database.Initialize(cfg); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize Routes
	api.SetupRoutes(router)

	// Add Swagger route - make sure this is before router.Run
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start Server
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
