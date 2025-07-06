package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ad-tracking-system/internal/metrics"
	"ad-tracking-system/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

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

	go s.publishToKafka(clickEvent)

	c.JSON(http.StatusOK, gin.H{"status": "recorded"})
}

func (s *Server) publishToKafka(clickEvent models.ClickEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventBytes, err := json.Marshal(clickEvent)
	if err != nil {
		s.logger.WithError(err).Error("Failed to serialize click event")
		return
	}

	err = s.KafkaWriter.WriteMessages(
		ctx,
		kafka.Message{
			Key:   []byte(strconv.Itoa(int(clickEvent.AdID))),
			Value: eventBytes,
		},
	)
	if err != nil {
		s.logger.WithError(err).Error("Failed to publish click event to Kafka")
	}
}

func (s *Server) GetAnalytics(c *gin.Context) {
	start := time.Now()
	defer func() {
		metrics.ResponseTime.WithLabelValues("GET", "/ads/analytics", strconv.Itoa(c.Writer.Status())).Observe(time.Since(start).Seconds())
	}()

	adIDStr := c.Query("ad_id")
	timeframe := c.DefaultQuery("timeframe", "24h")

	duration := s.parseDuration(timeframe)
	since := time.Now().UTC().Add(-duration)

	// Use UTC for consistent timezone handling
	beginningOfToday := time.Date(time.Now().UTC().Year(), time.Now().UTC().Month(), time.Now().UTC().Day(), 0, 0, 0, 0, time.UTC)

	s.logger.WithFields(logrus.Fields{
		"timeframe": timeframe,
		"duration":  duration,
		"since":     since,
		"now":       time.Now().UTC(),
		"ad_id":     adIDStr,
	}).Info("Analytics request parameters")

	debugInfo := s.getDebugCounts(adIDStr, since, beginningOfToday)

	if adIDStr != "" {
		adID, err := strconv.ParseUint(adIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ad_id"})
			return
		}

		analytics := s.analyticsRepository.GetAdAnalytics(uint(adID), since)

		c.JSON(http.StatusOK, gin.H{
			"analytics": analytics,
			"debug":     debugInfo,
		})
	} else {
		analytics := s.analyticsRepository.GetAllAnalytics(since)

		c.JSON(http.StatusOK, gin.H{
			"analytics": analytics,
			"debug":     debugInfo,
		})
	}
}

func (s *Server) getDebugCounts(adIDStr string, since, beginningOfToday time.Time) gin.H {
	var totalCount int64
	s.db.Model(&models.ClickEvent{}).Count(&totalCount)

	var filteredCount int64
	var filteredCountToday int64

	// Add timezone-aware debugging
	var timezoneTestCount int64
	s.db.Raw("SELECT COUNT(*) FROM click_events WHERE timestamp AT TIME ZONE 'UTC' >= ? AT TIME ZONE 'UTC'", since).Scan(&timezoneTestCount)

	if adIDStr != "" {
		adID, err := strconv.ParseUint(adIDStr, 10, 32)
		if err != nil {
			return gin.H{"error": "Invalid ad_id"}
		}
		s.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", uint(adID), since).Count(&filteredCount)
		s.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", uint(adID), beginningOfToday).Count(&filteredCountToday)
	} else {
		s.db.Model(&models.ClickEvent{}).Where("timestamp >= ?", since).Count(&filteredCount)
		s.db.Model(&models.ClickEvent{}).Where("timestamp >= ?", beginningOfToday).Count(&filteredCountToday)
	}

	// Get sample timestamps for debugging
	var sampleTimestamps []time.Time
	s.db.Model(&models.ClickEvent{}).Select("timestamp").Order("timestamp desc").Limit(3).Pluck("timestamp", &sampleTimestamps)

	debugInfo := gin.H{
		"total_records":          totalCount,
		"filtered_records":       filteredCount,
		"filtered_records_today": filteredCountToday,
		"timezone_test_count":    timezoneTestCount,
		"since":                  since,
		"beginning_of_today":     beginningOfToday,
		"now":                    time.Now().UTC(),
		"sample_timestamps":      sampleTimestamps,
	}

	s.logger.WithFields(logrus.Fields{
		"total_count":          totalCount,
		"filtered_count":       filteredCount,
		"filtered_count_today": filteredCountToday,
		"timezone_test_count":  timezoneTestCount,
		"since":                since,
		"beginning_of_today":   beginningOfToday,
		"sample_timestamps":    sampleTimestamps,
	}).Info("Record counts with timezone debugging")

	return debugInfo
}

func (s *Server) parseDuration(timeframe string) time.Duration {
	switch timeframe {
	case "1h":
		return time.Hour
	case "24h":
		return 24 * time.Hour
	case "7d":
		return 7 * 24 * time.Hour
	case "all":
		return 10 * 365 * 24 * time.Hour // 10 years
	default:
		return 24 * time.Hour
	}
}

func (s *Server) DebugAnalytics(c *gin.Context) {
	adIDStr := c.Query("ad_id")
	timeframe := c.DefaultQuery("timeframe", "24h")

	duration := s.parseDuration(timeframe)
	since := time.Now().UTC().Add(-duration)

	// Get sample data to understand what's in the database
	var clickEvents []models.ClickEvent
	query := s.db.Order("timestamp DESC").Limit(10)

	if adIDStr != "" {
		adID, err := strconv.ParseUint(adIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ad_id"})
			return
		}
		query = query.Where("ad_id = ?", uint(adID))
	}

	query.Find(&clickEvents)

	// Get counts with different approaches
	var totalCount int64
	var sinceCount int64
	var recentCount int64

	s.db.Model(&models.ClickEvent{}).Count(&totalCount)
	s.db.Model(&models.ClickEvent{}).Where("timestamp >= ?", since).Count(&sinceCount)
	s.db.Model(&models.ClickEvent{}).Where("timestamp >= ?", time.Now().UTC().Add(-time.Hour)).Count(&recentCount)

	// Test raw SQL approach
	var rawResult struct {
		TotalClicks int64 `db:"total_clicks"`
		LastHour    int64 `db:"last_hour"`
		LastDay     int64 `db:"last_day"`
	}

	lastHour := time.Now().UTC().Add(-time.Hour)
	lastDay := time.Now().UTC().Add(-24 * time.Hour)

	if adIDStr != "" {
		adID, _ := strconv.ParseUint(adIDStr, 10, 32)
		s.db.Raw(`
			SELECT 
				COUNT(*) as total_clicks,
				COUNT(CASE WHEN timestamp >= ? THEN 1 END) as last_hour,
				COUNT(CASE WHEN timestamp >= ? THEN 1 END) as last_day
			FROM click_events 
			WHERE ad_id = ?
		`, lastHour, lastDay, uint(adID)).Scan(&rawResult)
	} else {
		s.db.Raw(`
			SELECT 
				COUNT(*) as total_clicks,
				COUNT(CASE WHEN timestamp >= ? THEN 1 END) as last_hour,
				COUNT(CASE WHEN timestamp >= ? THEN 1 END) as last_day
			FROM click_events
		`, lastHour, lastDay).Scan(&rawResult)
	}

	// Test analytics repository
	var analyticsResult interface{}
	if adIDStr != "" {
		adID, _ := strconv.ParseUint(adIDStr, 10, 32)
		analyticsResult = s.analyticsRepository.GetAdAnalytics(uint(adID), since)
	} else {
		analyticsResult = s.analyticsRepository.GetAllAnalytics(since)
	}

	c.JSON(http.StatusOK, gin.H{
		"debug_info": gin.H{
			"query_params": gin.H{
				"ad_id":     adIDStr,
				"timeframe": timeframe,
				"duration":  duration.String(),
				"since":     since,
			},
			"current_time": gin.H{
				"utc":       time.Now().UTC(),
				"local":     time.Now(),
				"last_hour": lastHour,
				"last_day":  lastDay,
			},
			"counts": gin.H{
				"total_in_db":    totalCount,
				"since_count":    sinceCount,
				"recent_count":   recentCount,
				"raw_sql_result": rawResult,
			},
			"sample_records": clickEvents,
		},
		"analytics_result": analyticsResult,
	})
}

func (s *Server) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"version":   "1.0.0",
	})
}
