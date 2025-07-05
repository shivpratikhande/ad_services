package services

import (
	"context"
	"time"

	"ad-tracking-system/internal/metrics"
	"ad-tracking-system/internal/models"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ClickQueue struct {
	events chan models.ClickEvent
	db     *gorm.DB
	logger *logrus.Logger
}

func NewClickQueue(db *gorm.DB, logger *logrus.Logger, bufferSize int) *ClickQueue {
	return &ClickQueue{
		events: make(chan models.ClickEvent, bufferSize),
		db:     db,
		logger: logger,
	}
}

func (q *ClickQueue) Enqueue(event models.ClickEvent) bool {
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
	batch := make([]models.ClickEvent, 0, batchSize)
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

func (q *ClickQueue) processBatch(events []models.ClickEvent) {
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
		metrics.ClicksProcessed.Add(float64(len(events)))
		break
	}
}

func (q *ClickQueue) GetEvents() chan models.ClickEvent {
	return q.events
}
