package repository

import (
	"time"

	"ad-tracking-system/internal/models"

	"gorm.io/gorm"
)

type AnalyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) GetAdAnalytics(adID uint, since time.Time) models.AnalyticsResponse {
	var total int64
	var lastHour int64
	var lastDay int64

	// Use UTC time for consistency
	now := time.Now().UTC()
	sinceUTC := since.UTC()

	// Total clicks since the specified time
	r.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", adID, sinceUTC).Count(&total)

	// Last hour
	r.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", adID, now.Add(-time.Hour)).Count(&lastHour)

	// Last day
	r.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", adID, now.Add(-24*time.Hour)).Count(&lastDay)

	return models.AnalyticsResponse{
		AdID:       adID,
		ClickCount: total,
		LastHour:   lastHour,
		LastDay:    lastDay,
	}
}

func (r *AnalyticsRepository) GetAllAnalytics(since time.Time) []models.AnalyticsResponse {
	var results []struct {
		AdID       uint  `json:"ad_id"`
		ClickCount int64 `json:"click_count"`
	}

	sinceUTC := since.UTC()

	// Use Scan instead of Find for better compatibility with GROUP BY
	err := r.db.Model(&models.ClickEvent{}).
		Select("ad_id, count(*) as click_count").
		Where("timestamp >= ?", sinceUTC).
		Group("ad_id").
		Scan(&results).Error

	if err != nil {
		// Log the error if you have logging available
		// return empty slice if query fails
		return []models.AnalyticsResponse{}
	}

	// If no results found, return empty slice instead of nil
	if len(results) == 0 {
		return []models.AnalyticsResponse{}
	}

	analytics := make([]models.AnalyticsResponse, len(results))
	for i, result := range results {
		analytics[i] = r.GetAdAnalytics(result.AdID, since)
	}

	return analytics
}

// Alternative simpler version for GetAllAnalytics that doesn't call GetAdAnalytics recursively
func (r *AnalyticsRepository) GetAllAnalyticsSimple(since time.Time) []models.AnalyticsResponse {
	var results []struct {
		AdID       uint  `json:"ad_id"`
		ClickCount int64 `json:"click_count"`
	}

	sinceUTC := since.UTC()
	now := time.Now().UTC()

	// Get basic counts
	err := r.db.Model(&models.ClickEvent{}).
		Select("ad_id, count(*) as click_count").
		Where("timestamp >= ?", sinceUTC).
		Group("ad_id").
		Scan(&results).Error

	if err != nil || len(results) == 0 {
		return []models.AnalyticsResponse{}
	}

	analytics := make([]models.AnalyticsResponse, len(results))
	for i, result := range results {
		// Calculate last hour and last day for each ad
		var lastHour, lastDay int64
		r.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", result.AdID, now.Add(-time.Hour)).Count(&lastHour)
		r.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", result.AdID, now.Add(-24*time.Hour)).Count(&lastDay)

		analytics[i] = models.AnalyticsResponse{
			AdID:       result.AdID,
			ClickCount: result.ClickCount,
			LastHour:   lastHour,
			LastDay:    lastDay,
		}
	}

	return analytics
}
