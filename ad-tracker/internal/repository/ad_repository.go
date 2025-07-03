package repository

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"ad-tracker/internal/models"

	_ "github.com/lib/pq"
)

type AdRepository struct {
	db *sql.DB
}

func NewAdRepository(db *sql.DB) *AdRepository {
	return &AdRepository{db: db}
}

func (r *AdRepository) CreateAdEvent(event *models.AdEventRequest) (*models.AdEvent, error) {
	var metadataJSON []byte
	var err error

	if event.Metadata != nil {
		metadataJSON, err = json.Marshal(event.Metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
		INSERT INTO ads (campaign_id, ad_group_id, ad_id, user_id, event_type, metadata, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, timestamp, created_at
	`

	adEvent := &models.AdEvent{
		CampaignID: event.CampaignID,
		AdGroupID:  event.AdGroupID,
		AdID:       event.AdID,
		UserID:     event.UserID,
		EventType:  event.EventType,
		Metadata:   event.Metadata,
		Timestamp:  time.Now(),
	}

	err = r.db.QueryRow(
		query,
		event.CampaignID,
		event.AdGroupID,
		event.AdID,
		event.UserID,
		event.EventType,
		metadataJSON,
		adEvent.Timestamp,
	).Scan(&adEvent.ID, &adEvent.Timestamp, &adEvent.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create ad event: %w", err)
	}

	return adEvent, nil
}

func (r *AdRepository) GetAdEvents(campaignID string, limit int, offset int) ([]models.AdEvent, error) {
	query := `
		SELECT id, campaign_id, ad_group_id, ad_id, user_id, event_type, timestamp, metadata, created_at
		FROM ads
		WHERE campaign_id = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, campaignID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query ad events: %w", err)
	}
	defer rows.Close()

	var events []models.AdEvent
	for rows.Next() {
		var event models.AdEvent
		var metadataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.CampaignID,
			&event.AdGroupID,
			&event.AdID,
			&event.UserID,
			&event.EventType,
			&event.Timestamp,
			&metadataJSON,
			&event.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan ad event: %w", err)
		}

		if metadataJSON != nil {
			err = json.Unmarshal(metadataJSON, &event.Metadata)
			if err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		events = append(events, event)
	}

	return events, nil
}

func (r *AdRepository) GetCampaignSummary(campaignID string) (*models.CampaignSummary, error) {
	query := `
		SELECT 
			campaign_id,
			SUM(CASE WHEN event_type = 'impression' THEN 1 ELSE 0 END) as impressions,
			SUM(CASE WHEN event_type = 'click' THEN 1 ELSE 0 END) as clicks,
			SUM(CASE WHEN event_type = 'conversion' THEN 1 ELSE 0 END) as conversions
		FROM ads
		WHERE campaign_id = $1
		GROUP BY campaign_id
	`

	var summary models.CampaignSummary
	err := r.db.QueryRow(query, campaignID).Scan(
		&summary.CampaignID,
		&summary.Impressions,
		&summary.Clicks,
		&summary.Conversions,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return &models.CampaignSummary{CampaignID: campaignID}, nil
		}
		return nil, fmt.Errorf("failed to get campaign summary: %w", err)
	}

	// Calculate rates
	if summary.Impressions > 0 {
		summary.CTR = float64(summary.Clicks) / float64(summary.Impressions) * 100
	}
	if summary.Clicks > 0 {
		summary.CVR = float64(summary.Conversions) / float64(summary.Clicks) * 100
	}

	return &summary, nil
}

func (r *AdRepository) GetAnalytics(campaignID string, days int) ([]models.AdAnalytics, error) {
	query := `
		SELECT campaign_id, ad_group_id, ad_id, event_type, event_count, event_date
		FROM ad_analytics
		WHERE campaign_id = $1 AND event_date >= $2
		ORDER BY event_date DESC, event_type
	`

	startDate := time.Now().AddDate(0, 0, -days)
	rows, err := r.db.Query(query, campaignID, startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query analytics: %w", err)
	}
	defer rows.Close()

	var analytics []models.AdAnalytics
	for rows.Next() {
		var analytic models.AdAnalytics
		err := rows.Scan(
			&analytic.CampaignID,
			&analytic.AdGroupID,
			&analytic.AdID,
			&analytic.EventType,
			&analytic.EventCount,
			&analytic.EventDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan analytics: %w", err)
		}
		analytics = append(analytics, analytic)
	}

	return analytics, nil
}
