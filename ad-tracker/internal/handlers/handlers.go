package handlers

import (
	"net/http"
	"strconv"
	"time"

	"ad-tracking-system/internal/metrics"
	"ad-tracking-system/internal/models"
	"ad-tracking-system/internal/repository"
	"ad-tracking-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Server struct {
	db                  *gorm.DB
	logger              *logrus.Logger
	clickQueue          *services.ClickQueue
	analyticsRepository *repository.AnalyticsRepository
}

func NewServer(db *gorm.DB, logger *logrus.Logger) *Server {
	clickQueue := services.NewClickQueue(db, logger, 10000)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	return &Server{
		db:                  db,
		logger:              logger,
		clickQueue:          clickQueue,
		analyticsRepository: analyticsRepo,
	}
}

func (s *Server) GetClickQueue() *services.ClickQueue {
	return s.clickQueue
}

func (s *Server) GetAds(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.ResponseTime.WithLabelValues("GET", "/ads", strconv.Itoa(c.Writer.Status())).Observe(time.Since(start).Seconds())
	}()

	var ads []models.Ad
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
		metrics.ResponseTime.WithLabelValues("POST", "/ads/click", strconv.Itoa(c.Writer.Status())).Observe(time.Since(start).Seconds())
	}()

	var req models.ClickRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate ad exists
	var ad models.Ad
	if err := s.db.First(&ad, req.AdID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ad not found"})
		return
	}

	// Create click event
	clickEvent := models.ClickEvent{
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
	metrics.ClicksReceived.WithLabelValues(strconv.FormatUint(uint64(req.AdID), 10)).Inc()
	metrics.QueueSize.Set(float64(len(s.clickQueue.GetEvents())))

	// Return immediately to client
	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

func (s *Server) GetAnalytics(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.ResponseTime.WithLabelValues("GET", "/ads/analytics", strconv.Itoa(c.Writer.Status())).Observe(time.Since(start).Seconds())
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

		analytics := s.analyticsRepository.GetAdAnalytics(uint(adID), since)
		c.JSON(http.StatusOK, analytics)
	} else {
		// Analytics for all ads
		analytics := s.analyticsRepository.GetAllAnalytics(since)
		c.JSON(http.StatusOK, gin.H{"analytics": analytics})
	}
}

func (s *Server) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}
