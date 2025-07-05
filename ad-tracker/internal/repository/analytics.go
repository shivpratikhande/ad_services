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

	// Total clicks
	r.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", adID, since).Count(&total)

	// Last hour
	r.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", adID, time.Now().Add(-time.Hour)).Count(&lastHour)

	// Last day
	r.db.Model(&models.ClickEvent{}).Where("ad_id = ? AND timestamp >= ?", adID, time.Now().Add(-24*time.Hour)).Count(&lastDay)

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

	r.db.Model(&models.ClickEvent{}).
		Select("ad_id, count(*) as click_count").
		Where("timestamp >= ?", since).
		Group("ad_id").
		Find(&results)

	analytics := make([]models.AnalyticsResponse, len(results))
	for i, result := range results {
		analytics[i] = r.GetAdAnalytics(result.AdID, since)
	}

	return analytics
}
