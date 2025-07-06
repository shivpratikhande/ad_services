package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ad-tracking-system/internal/metrics"
	"ad-tracking-system/internal/models"
	"ad-tracking-system/internal/repository"
	"ad-tracking-system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Server struct {
	db                  *gorm.DB
	logger              *logrus.Logger
	clickQueue          *services.ClickQueue
	analyticsRepository *repository.AnalyticsRepository
	KafkaWriter         *kafka.Writer
}

func NewServer(db *gorm.DB, logger *logrus.Logger, kafkaWriter *kafka.Writer) *Server {
	clickQueue := services.NewClickQueue(db, logger, 10000)
	analyticsRepo := repository.NewAnalyticsRepository(db)
	return &Server{
		db:                  db,
		logger:              logger,
		clickQueue:          clickQueue,
		analyticsRepository: analyticsRepo,
		KafkaWriter:         kafkaWriter,
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

	var ad models.Ad
	if err := s.db.First(&ad, req.AdID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ad not found"})
		return
	}

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

	if !s.clickQueue.Enqueue(clickEvent) {
		if err := s.db.Create(&clickEvent).Error; err != nil {
			s.logger.WithError(err).Error("Failed to save click event")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record click"})
			return
		}
	}

	metrics.ClicksReceived.WithLabelValues(strconv.FormatUint(uint64(req.AdID), 10)).Inc()
	metrics.QueueSize.Set(float64(len(s.clickQueue.GetEvents())))

	// Kafka: serialize and publish the event
	eventBytes, err := json.Marshal(clickEvent)
	if err != nil {
		s.logger.WithError(err).Error("Failed to serialize click event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	err = s.KafkaWriter.WriteMessages(
		c,
		kafka.Message{
			Key:   []byte(strconv.Itoa(int(clickEvent.AdID))),
			Value: eventBytes,
		},
	)
	if err != nil {
		s.logger.WithError(err).Error("Failed to publish click event to Kafka")
	}

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
	case "all":
		duration = 10 * 365 * 24 * time.Hour // 10 years
	default:
		duration = 24 * time.Hour
	}

	// Use UTC time for consistency with database
	since := time.Now().UTC().Add(-duration)

	// For debugging: also try beginning of today
	beginningOfToday := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)

	// Debug logging
	s.logger.WithFields(logrus.Fields{
		"timeframe": timeframe,
		"duration":  duration,
		"since":     since,
		"now":       time.Now().UTC(),
		"ad_id":     adIDStr,
	}).Info("Analytics request parameters")

	// Quick debug: count total records
	var totalCount int64
	s.db.Model(&models.ClickEvent{}).Count(&totalCount)

	var filteredCount int64
	var filteredCountToday int64
	if adIDStr != "" {
		adID, err := strconv.ParseUint(adIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ad_id"})
			return
		}
		s.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", uint(adID), since).Count(&filteredCount)
		s.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", uint(adID), beginningOfToday).Count(&filteredCountToday)
	} else {
		s.db.Model(&models.ClickEvent{}).Where("timestamp >= ?", since).Count(&filteredCount)
		s.db.Model(&models.ClickEvent{}).Where("timestamp >= ?", beginningOfToday).Count(&filteredCountToday)
	}

	s.logger.WithFields(logrus.Fields{
		"total_count":          totalCount,
		"filtered_count":       filteredCount,
		"filtered_count_today": filteredCountToday,
		"since":                since,
		"beginning_of_today":   beginningOfToday,
	}).Info("Record counts")

	if adIDStr != "" {
		// Analytics for specific ad
		adID, err := strconv.ParseUint(adIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ad_id"})
			return
		}

		analytics := s.analyticsRepository.GetAdAnalytics(uint(adID), since)

		// Add debug info
		c.JSON(http.StatusOK, gin.H{
			"analytics": analytics,
			"debug": gin.H{
				"total_records":          totalCount,
				"filtered_records":       filteredCount,
				"filtered_records_today": filteredCountToday,
				"since":                  since,
				"beginning_of_today":     beginningOfToday,
				"now":                    time.Now().UTC(),
			},
		})
	} else {
		// Analytics for all ads
		analytics := s.analyticsRepository.GetAllAnalytics(since)

		// Added debug info
		c.JSON(http.StatusOK, gin.H{
			"analytics": analytics,
			"debug": gin.H{
				"total_records":          totalCount,
				"filtered_records":       filteredCount,
				"filtered_records_today": filteredCountToday,
				"since":                  since,
				"beginning_of_today":     beginningOfToday,
				"now":                    time.Now().UTC(),
			},
		})
	}
}

func (s *Server) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}
