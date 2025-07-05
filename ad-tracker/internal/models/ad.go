package models

import "time"

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
