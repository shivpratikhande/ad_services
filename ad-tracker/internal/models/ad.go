package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventImpression EventType = "impression"
	EventClick      EventType = "click"
	EventConversion EventType = "conversion"
)

type AdEvent struct {
	ID         uuid.UUID              `json:"id" db:"id"`
	CampaignID string                 `json:"campaign_id" db:"campaign_id" binding:"required"`
	AdGroupID  string                 `json:"ad_group_id" db:"ad_group_id" binding:"required"`
	AdID       string                 `json:"ad_id" db:"ad_id" binding:"required"`
	UserID     *string                `json:"user_id,omitempty" db:"user_id"`
	EventType  EventType              `json:"event_type" db:"event_type" binding:"required,oneof=impression click conversion"`
	Timestamp  time.Time              `json:"timestamp" db:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

type AdEventRequest struct {
	CampaignID string                 `json:"campaign_id" binding:"required"`
	AdGroupID  string                 `json:"ad_group_id" binding:"required"`
	AdID       string                 `json:"ad_id" binding:"required"`
	UserID     *string                `json:"user_id,omitempty"`
	EventType  EventType              `json:"event_type" binding:"required,oneof=impression click conversion"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type AdAnalytics struct {
	CampaignID string    `json:"campaign_id" db:"campaign_id"`
	AdGroupID  string    `json:"ad_group_id" db:"ad_group_id"`
	AdID       string    `json:"ad_id" db:"ad_id"`
	EventType  EventType `json:"event_type" db:"event_type"`
	EventCount int       `json:"event_count" db:"event_count"`
	EventDate  time.Time `json:"event_date" db:"event_date"`
}

type CampaignSummary struct {
	CampaignID  string  `json:"campaign_id"`
	Impressions int     `json:"impressions"`
	Clicks      int     `json:"clicks"`
	Conversions int     `json:"conversions"`
	CTR         float64 `json:"ctr"` // Click-through rate
	CVR         float64 `json:"cvr"` // Conversion rate
}

// Helper method to convert metadata to JSON
func (a *AdEvent) MetadataJSON() ([]byte, error) {
	if a.Metadata == nil {
		return nil, nil
	}
	return json.Marshal(a.Metadata)
}
