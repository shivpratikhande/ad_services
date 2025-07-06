package repositories

import (
	"ad-tracking-system/internal/models"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AnalyticsRepository struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewAnalyticsRepository(db *gorm.DB, logger *logrus.Logger) *AnalyticsRepository {
	return &AnalyticsRepository{
		db:     db,
		logger: logger,
	}
}

func (r *AnalyticsRepository) GetAdAnalytics(adID uint, since time.Time) models.AnalyticsResponse {
	var analytics models.AnalyticsResponse

	// Get basic click count for the timeframe
	var clickCount int64
	err := r.db.Model(&models.ClickEvent{}).
		Where("ad_id = ? AND timestamp >= ?", adID, since).
		Count(&clickCount).Error

	if err != nil {
		r.logger.WithError(err).Error("Failed to get click count")
		return models.AnalyticsResponse{AdID: adID}
	}

	// Get last hour count
	lastHour := time.Now().UTC().Add(-time.Hour)
	var lastHourCount int64
	err = r.db.Model(&models.ClickEvent{}).
		Where("ad_id = ? AND timestamp >= ?", adID, lastHour).
		Count(&lastHourCount).Error

	if err != nil {
		r.logger.WithError(err).Error("Failed to get last hour count")
	}

	// Get last day count
	lastDay := time.Now().UTC().Add(-24 * time.Hour)
	var lastDayCount int64
	err = r.db.Model(&models.ClickEvent{}).
		Where("ad_id = ? AND timestamp >= ?", adID, lastDay).
		Count(&lastDayCount).Error

	if err != nil {
		r.logger.WithError(err).Error("Failed to get last day count")
	}

	analytics.AdID = adID
	analytics.ClickCount = clickCount
	analytics.LastHour = lastHourCount
	analytics.LastDay = lastDayCount
	// CTR would need impression data to calculate, leaving it as 0 for now

	r.logger.WithFields(logrus.Fields{
		"ad_id":       adID,
		"click_count": clickCount,
		"last_hour":   lastHourCount,
		"last_day":    lastDayCount,
		"since":       since,
	}).Info("Retrieved ad analytics")

	return analytics
}

func (r *AnalyticsRepository) GetAllAnalytics(since time.Time) []models.AnalyticsResponse {
	var allAnalytics []models.AnalyticsResponse

	// Get all unique ad IDs that have clicks since the specified time
	var adIDs []uint
	err := r.db.Model(&models.ClickEvent{}).
		Where("timestamp >= ?", since).
		Distinct("ad_id").
		Pluck("ad_id", &adIDs).Error

	if err != nil {
		r.logger.WithError(err).Error("Failed to get unique ad IDs")
		return allAnalytics
	}

	r.logger.WithFields(logrus.Fields{
		"ad_ids": adIDs,
		"since":  since,
	}).Info("Found ad IDs for analytics")

	// Get analytics for each ad
	for _, adID := range adIDs {
		analytics := r.GetAdAnalytics(adID, since)
		allAnalytics = append(allAnalytics, analytics)
	}

	return allAnalytics
}

// Alternative method using raw SQL to handle potential timezone issues
func (r *AnalyticsRepository) GetAdAnalyticsWithRawSQL(adID uint, since time.Time) models.AnalyticsResponse {
	var analytics models.AnalyticsResponse

	lastHour := time.Now().UTC().Add(-time.Hour)
	lastDay := time.Now().UTC().Add(-24 * time.Hour)

	// Use a single query to get all counts
	var result struct {
		TotalClicks int64 `db:"total_clicks"`
		LastHour    int64 `db:"last_hour"`
		LastDay     int64 `db:"last_day"`
	}

	query := `
		SELECT 
			COUNT(*) as total_clicks,
			COUNT(CASE WHEN timestamp >= ? THEN 1 END) as last_hour,
			COUNT(CASE WHEN timestamp >= ? THEN 1 END) as last_day
		FROM click_events 
		WHERE ad_id = ? 
		AND timestamp >= ?
	`

	err := r.db.Raw(query, lastHour, lastDay, adID, since).Scan(&result).Error
	if err != nil {
		r.logger.WithError(err).Error("Failed to execute raw SQL analytics query")
		return models.AnalyticsResponse{AdID: adID}
	}

	analytics.AdID = adID
	analytics.ClickCount = result.TotalClicks
	analytics.LastHour = result.LastHour
	analytics.LastDay = result.LastDay

	r.logger.WithFields(logrus.Fields{
		"ad_id":       adID,
		"click_count": result.TotalClicks,
		"last_hour":   result.LastHour,
		"last_day":    result.LastDay,
		"method":      "raw_sql",
	}).Info("Retrieved ad analytics using raw SQL")

	return analytics
}

func (r *AnalyticsRepository) GetAllAnalyticsWithRawSQL(since time.Time) []models.AnalyticsResponse {
	var allAnalytics []models.AnalyticsResponse

	lastHour := time.Now().UTC().Add(-time.Hour)
	lastDay := time.Now().UTC().Add(-24 * time.Hour)

	// Get analytics for all ads in a single query
	var results []struct {
		AdID        uint  `db:"ad_id"`
		TotalClicks int64 `db:"total_clicks"`
		LastHour    int64 `db:"last_hour"`
		LastDay     int64 `db:"last_day"`
	}

	query := `
		SELECT 
			ad_id,
			COUNT(*) as total_clicks,
			COUNT(CASE WHEN timestamp >= ? THEN 1 END) as last_hour,
			COUNT(CASE WHEN timestamp >= ? THEN 1 END) as last_day
		FROM click_events 
		WHERE timestamp >= ?
		GROUP BY ad_id
		ORDER BY ad_id
	`

	err := r.db.Raw(query, lastHour, lastDay, since).Scan(&results).Error
	if err != nil {
		r.logger.WithError(err).Error("Failed to execute raw SQL analytics query for all ads")
		return allAnalytics
	}

	// Convert results to AnalyticsResponse
	for _, result := range results {
		analytics := models.AnalyticsResponse{
			AdID:       result.AdID,
			ClickCount: result.TotalClicks,
			LastHour:   result.LastHour,
			LastDay:    result.LastDay,
		}
		allAnalytics = append(allAnalytics, analytics)
	}

	r.logger.WithFields(logrus.Fields{
		"results_count": len(allAnalytics),
		"since":         since,
		"method":        "raw_sql",
	}).Info("Retrieved all analytics using raw SQL")

	return allAnalytics
}
