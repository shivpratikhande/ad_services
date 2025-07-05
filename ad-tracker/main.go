package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Models
type Ad struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	ImageURL  string    `json:"image_url" gorm:"not null"`
	TargetURL string    `json:"target_url" gorm:"not null"`
	Title     string    `json:"title"`
	Active    bool      `json:"active" gorm:"default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ClickEvent struct {
	ID                uint      `json:"id" gorm:"primaryKey"`
	AdID              uint      `json:"ad_id" gorm:"not null;index"`
	Timestamp         time.Time `json:"timestamp" gorm:"not null;index"`
	IPAddress         string    `json:"ip_address"`
	VideoPlaybackTime int64     `json:"video_playback_time"` // in seconds
	UserAgent         string    `json:"user_agent"`
	Processed         bool      `json:"processed" gorm:"default:false;index"`
	CreatedAt         time.Time `json:"created_at"`
}

type ClickRequest struct {
	AdID              uint  `json:"ad_id" binding:"required"`
	Timestamp         int64 `json:"timestamp"`
	VideoPlaybackTime int64 `json:"video_playback_time"`
}

type AnalyticsResponse struct {
	AdID       uint    `json:"ad_id"`
	ClickCount int64   `json:"click_count"`
	CTR        float64 `json:"ctr,omitempty"`
	LastHour   int64   `json:"last_hour"`
	LastDay    int64   `json:"last_day"`
}

// Queue for async processing
type ClickQueue struct {
	events chan ClickEvent
	db     *gorm.DB
	logger *logrus.Logger
}

func NewClickQueue(db *gorm.DB, logger *logrus.Logger, bufferSize int) *ClickQueue {
	return &ClickQueue{
		events: make(chan ClickEvent, bufferSize),
		db:     db,
		logger: logger,
	}
}

func (q *ClickQueue) Enqueue(event ClickEvent) bool {
	select {
	case q.events <- event:
		return true
	default:
		// Queue is full, handle gracefully
		q.logger.Warn("Click queue is full, dropping event")
		return false
	}
}

func (q *ClickQueue) StartProcessor(ctx context.Context) {
	batchSize := 100
	batchTimeout := 5 * time.Second
	batch := make([]ClickEvent, 0, batchSize)
	timer := time.NewTimer(batchTimeout)

	for {
		select {
		case <-ctx.Done():
			// Process remaining events
			if len(batch) > 0 {
				q.processBatch(batch)
			}
			return
		case event := <-q.events:
			batch = append(batch, event)
			if len(batch) >= batchSize {
				q.processBatch(batch)
				batch = batch[:0]
				timer.Reset(batchTimeout)
			}
		case <-timer.C:
			if len(batch) > 0 {
				q.processBatch(batch)
				batch = batch[:0]
			}
			timer.Reset(batchTimeout)
		}
	}
}

func (q *ClickQueue) processBatch(events []ClickEvent) {
	if len(events) == 0 {
		return
	}

	// Batch insert with retry logic
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := q.db.Create(&events).Error; err != nil {
			q.logger.WithError(err).Warnf("Failed to insert batch (attempt %d/%d)", i+1, maxRetries)
			if i == maxRetries-1 {
				q.logger.WithError(err).Error("Failed to insert click events after all retries")
				// Could implement dead letter queue here
			}
			time.Sleep(time.Duration(i+1) * time.Second)
			continue
		}
		clicksProcessed.Add(float64(len(events)))
		break
	}
}

// Prometheus metrics
var (
	clicksReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ad_clicks_received_total",
			Help: "Total number of click events received",
		},
		[]string{"ad_id"},
	)

	clicksProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ad_clicks_processed_total",
			Help: "Total number of click events processed",
		},
	)

	responseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	queueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "click_queue_size",
			Help: "Current size of the click processing queue",
		},
	)
)

func init() {
	prometheus.MustRegister(clicksReceived)
	prometheus.MustRegister(clicksProcessed)
	prometheus.MustRegister(responseTime)
	prometheus.MustRegister(queueSize)
}

// Server struct
type Server struct {
	db         *gorm.DB
	logger     *logrus.Logger
	clickQueue *ClickQueue
}

func NewServer(db *gorm.DB, logger *logrus.Logger) *Server {
	clickQueue := NewClickQueue(db, logger, 10000)
	return &Server{
		db:         db,
		logger:     logger,
		clickQueue: clickQueue,
	}
}

// Handlers
func (s *Server) GetAds(c *gin.Context) {
	start := time.Now()
	defer func() {
		responseTime.WithLabelValues("GET", "/ads", strconv.Itoa(c.Writer.Status())).Observe(time.Since(start).Seconds())
	}()

	var ads []Ad
	if err := s.db.Where("active = ?", true).Find(&ads).Error; err != nil {
		s.logger.WithError(err).Error("Failed to fetch ads")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ads"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ads": ads})
}

func (s *Server) PostClick(c *gin.Context) {
	start := time.Now()
	defer func() {
		responseTime.WithLabelValues("POST", "/ads/click", strconv.Itoa(c.Writer.Status())).Observe(time.Since(start).Seconds())
	}()

	var req ClickRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate ad exists
	var ad Ad
	if err := s.db.First(&ad, req.AdID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ad not found"})
		return
	}

	// Create click event
	clickEvent := ClickEvent{
		AdID:              req.AdID,
		Timestamp:         time.Now(),
		IPAddress:         c.ClientIP(),
		VideoPlaybackTime: req.VideoPlaybackTime,
		UserAgent:         c.GetHeader("User-Agent"),
	}

	if req.Timestamp > 0 {
		clickEvent.Timestamp = time.Unix(req.Timestamp, 0)
	}

	// Enqueue for async processing
	if !s.clickQueue.Enqueue(clickEvent) {
		// Fallback to synchronous processing if queue is full
		if err := s.db.Create(&clickEvent).Error; err != nil {
			s.logger.WithError(err).Error("Failed to save click event")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record click"})
			return
		}
	}

	// Update metrics
	clicksReceived.WithLabelValues(strconv.FormatUint(uint64(req.AdID), 10)).Inc()
	queueSize.Set(float64(len(s.clickQueue.events)))

	// Return immediately to client
	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

func (s *Server) GetAnalytics(c *gin.Context) {
	start := time.Now()
	defer func() {
		responseTime.WithLabelValues("GET", "/ads/analytics", strconv.Itoa(c.Writer.Status())).Observe(time.Since(start).Seconds())
	}()

	adIDStr := c.Query("ad_id")
	timeframe := c.DefaultQuery("timeframe", "24h")

	var duration time.Duration
	switch timeframe {
	case "1h":
		duration = time.Hour
	case "24h":
		duration = 24 * time.Hour
	case "7d":
		duration = 7 * 24 * time.Hour
	default:
		duration = 24 * time.Hour
	}

	since := time.Now().Add(-duration)

	if adIDStr != "" {
		// Analytics for specific ad
		adID, err := strconv.ParseUint(adIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ad_id"})
			return
		}

		analytics := s.getAdAnalytics(uint(adID), since)
		c.JSON(http.StatusOK, analytics)
	} else {
		// Analytics for all ads
		analytics := s.getAllAnalytics(since)
		c.JSON(http.StatusOK, gin.H{"analytics": analytics})
	}
}

func (s *Server) getAdAnalytics(adID uint, since time.Time) AnalyticsResponse {
	var total int64
	var lastHour int64
	var lastDay int64

	// Total clicks
	s.db.Model(&ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", adID, since).Count(&total)

	// Last hour
	s.db.Model(&ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", adID, time.Now().Add(-time.Hour)).Count(&lastHour)

	// Last day
	s.db.Model(&ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", adID, time.Now().Add(-24*time.Hour)).Count(&lastDay)

	return AnalyticsResponse{
		AdID:       adID,
		ClickCount: total,
		LastHour:   lastHour,
		LastDay:    lastDay,
	}
}

func (s *Server) getAllAnalytics(since time.Time) []AnalyticsResponse {
	var results []struct {
		AdID       uint  `json:"ad_id"`
		ClickCount int64 `json:"click_count"`
	}

	s.db.Model(&ClickEvent{}).
		Select("ad_id, count(*) as click_count").
		Where("timestamp >= ?", since).
		Group("ad_id").
		Find(&results)

	analytics := make([]AnalyticsResponse, len(results))
	for i, result := range results {
		analytics[i] = s.getAdAnalytics(result.AdID, since)
	}

	return analytics
}

// Middleware
func LoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		timestamp := time.Now()
		latency := timestamp.Sub(start)

		if raw != "" {
			path = path + "?" + raw
		}

		logger.WithFields(logrus.Fields{
			"status_code": c.Writer.Status(),
			"latency":     latency,
			"client_ip":   c.ClientIP(),
			"method":      c.Request.Method,
			"path":        path,
		}).Info("Request processed")
	}
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Utility functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setupDatabase() (*gorm.DB, error) {
	dsn := getEnv("DATABASE_URL", "postgresql://neondb_owner:npg_kGErW7FMByH2@ep-muddy-poetry-adb64k0i-pooler.c-2.us-east-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require")

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// Auto-migrate schemas
	if err := db.AutoMigrate(&Ad{}, &ClickEvent{}); err != nil {
		return nil, err
	}

	// Set connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

func setupLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	level := getEnv("LOG_LEVEL", "info")
	switch level {
	case "debug":
		logger.SetLevel(logrus.DebugLevel)
	case "warn":
		logger.SetLevel(logrus.WarnLevel)
	case "error":
		logger.SetLevel(logrus.ErrorLevel)
	default:
		logger.SetLevel(logrus.InfoLevel)
	}

	return logger
}

func seedDatabase(db *gorm.DB) error {
	// Check if we already have ads
	var count int64
	db.Model(&Ad{}).Count(&count)
	if count > 0 {
		return nil
	}

	// Create sample ads
	sampleAds := []Ad{
		{
			ImageURL:  "https://example.com/ad1.jpg",
			TargetURL: "https://example.com/product1",
			Title:     "Amazing Product 1",
			Active:    true,
		},
		{
			ImageURL:  "https://example.com/ad2.jpg",
			TargetURL: "https://example.com/product2",
			Title:     "Great Service 2",
			Active:    true,
		},
		{
			ImageURL:  "https://example.com/ad3.jpg",
			TargetURL: "https://example.com/product3",
			Title:     "Special Offer 3",
			Active:    true,
		},
	}

	return db.Create(&sampleAds).Error
}

func main() {
	// Setup logger
	logger := setupLogger()

	// Setup database
	db, err := setupDatabase()
	if err != nil {
		logger.WithError(err).Fatal("Failed to connect to database")
	}

	// Seed database with sample data
	if err := seedDatabase(db); err != nil {
		logger.WithError(err).Warn("Failed to seed database")
	}

	// Create server
	server := NewServer(db, logger)

	// Start click queue processor
	ctx, cancel := context.WithCancel(context.Background())
	go server.clickQueue.StartProcessor(ctx)

	// Setup Gin router
	if getEnv("GIN_MODE", "debug") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(LoggingMiddleware(logger))
	r.Use(CORSMiddleware())

	// API routes
	api := r.Group("/api/v1")
	{
		api.GET("/ads", server.GetAds)
		api.POST("/ads/click", server.PostClick)
		api.GET("/ads/analytics", server.GetAnalytics)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	// Metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Start server
	port := getEnv("PORT", "8080")
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	logger.WithField("port", port).Info("Server started")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Cancel context for click processor
	cancel()

	// Shutdown server
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.WithError(err).Fatal("Server forced to shutdown")
	}

	logger.Info("Server exited")
}
