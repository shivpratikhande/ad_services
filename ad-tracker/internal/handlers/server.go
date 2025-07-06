package handlers

import (
	repositories "ad-tracking-system/internal/repository"
	"ad-tracking-system/internal/services"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Server struct {
	db                  *gorm.DB
	logger              *logrus.Logger
	clickQueue          *services.ClickQueue
	analyticsRepository *repositories.AnalyticsRepository
	KafkaWriter         *kafka.Writer
}

func NewServer(db *gorm.DB, logger *logrus.Logger, kafkaWriter *kafka.Writer) *Server {
	clickQueue := services.NewClickQueue(db, logger, 10000)
	analyticsRepo := repositories.NewAnalyticsRepository(db, logger)

	// Start background flusher for the queue
	// clickQueue.StartBackgroundFlusher(30 * time.Second)

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

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	s.logger.Info("Shutting down server...")

	// Stop the background flusher and do final flush
	// s.clickQueue.Stop()

	s.logger.Info("Server shutdown complete")
}
