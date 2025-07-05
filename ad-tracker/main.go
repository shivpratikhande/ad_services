package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ad-tracking-system/internal/config"
	"ad-tracking-system/internal/database"
	"ad-tracking-system/internal/handlers"
	"ad-tracking-system/internal/logger"
	"ad-tracking-system/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Setup logger
	logLevel := config.GetEnv("LOG_LEVEL", "info")
	log := logger.SetupLogger(logLevel)

	// database connnection
	databaseURL := config.GetEnv("DATABASE_URL", "postgresql://neondb_owner:npg_2gSkEdIJryj9@ep-delicate-sea-a8qjn7u5-pooler.eastus2.azure.neon.tech/neondb?sslmode=require&channel_binding=require")
	db, err := database.SetupDatabase(databaseURL)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to database")
	}

	// Seed database with sample data
	if err := database.SeedDatabase(db); err != nil {
		log.WithError(err).Warn("Failed to seed database")
	}

	// Create server
	server := handlers.NewServer(db, log)

	// Start click queue processor
	ctx, cancel := context.WithCancel(context.Background())
	go server.GetClickQueue().StartProcessor(ctx)

	// Setup Gin router
	if config.GetEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.LoggingMiddleware(log))
	r.Use(middleware.CORSMiddleware())

	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/ads", server.GetAds)
		api.POST("/ads/click", server.PostClick)
		api.GET("/ads/analytics", server.GetAnalytics)
	}

	// Health check
	r.GET("/health", server.Health)

	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Start server
	port := config.GetEnv("PORT", "8080")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Failed to start server")
		}
	}()

	log.WithField("port", port).Info("Server started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Cancel context for click processor
	cancel()

	// Shutdown server
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.WithError(err).Fatal("Server forced to shutdown")
	}

	log.Info("Server exited")
}
