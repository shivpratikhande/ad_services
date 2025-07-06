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
	"github.com/segmentio/kafka-go"
)

func main() {
	// Setup logger
	logLevel := config.GetEnv("LOG_LEVEL", "info")
	log := logger.SetupLogger(logLevel)

	// Kafka configuration
	kafkaBroker := config.GetEnv("KAFKA_BROKER", "localhost:9092")
	kafkaTopic := config.GetEnv("KAFKA_TOPIC", "ad-events")

	// kafka new writer
	kafkaWriter := &kafka.Writer{
		Addr:         kafka.TCP(kafkaBroker),
		Topic:        kafkaTopic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
	}

	defer func() {
		if err := kafkaWriter.Close(); err != nil {
			log.WithError(err).Error("Failed to close Kafka writer")
		}
	}()

	// db connection
	databaseURL := config.GetEnv("DATABASE_URL", "postgres://user:password@localhost:5432/adtracker?sslmode=disable")
	db, err := database.SetupDatabase(databaseURL)
	if err != nil {
		log.WithError(err).Fatal("Failed to connect to database")
	}

	// feed db with sample data
	if err := database.SeedDatabase(db); err != nil {
		log.WithError(err).Warn("Failed to seed database")
	}

	server := handlers.NewServer(db, log, kafkaWriter)

	// Start click queue processor
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
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

	r.GET("/health", server.Health)

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	port := config.GetEnv("PORT", "8080")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("Failed to start server")
		}
	}()

	log.WithField("port", port).Info("Server started")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.WithError(err).Fatal("Server forced to shutdown")
	}

	log.Info("Server exited")
}
