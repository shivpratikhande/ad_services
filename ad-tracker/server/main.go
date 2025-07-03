package main

import (
	"database/sql"
	"log"
	"net/http"

	"ad-tracker/internal/config"
	"ad-tracker/internal/handlers"
	"ad-tracker/internal/repository"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	// Initialize repository and handlers
	adRepo := repository.NewAdRepository(db)
	adHandler := handlers.NewAdHandler(adRepo)

	// Setup Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Routes
	api := router.Group("/api/v1")
	{
		// Health check
		api.GET("/health", adHandler.HealthCheck)

		// Ad events
		api.POST("/events", adHandler.CreateAdEvent)
		api.GET("/campaigns/:campaignId/events", adHandler.GetAdEvents)
		api.GET("/campaigns/:campaignId/summary", adHandler.GetCampaignSummary)
		api.GET("/campaigns/:campaignId/analytics", adHandler.GetAnalytics)
	}

	// Start server
	log.Printf("Server starting on port %s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
