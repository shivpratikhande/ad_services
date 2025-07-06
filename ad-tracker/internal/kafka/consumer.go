package kafka

import (
	"context"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

type Consumer struct {
	reader *kafka.Reader
	logger *logrus.Logger
}

func NewConsumer(brokerURL, topic, groupID string, logger *logrus.Logger) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{brokerURL},
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})

	return &Consumer{
		reader: reader,
		logger: logger,
	}
}

func (c *Consumer) ReadMessage(ctx context.Context) (kafka.Message, error) {
	message, err := c.reader.ReadMessage(ctx)
	if err != nil {
		c.logger.WithError(err).Error("Failed to read message from Kafka")
		return kafka.Message{}, fmt.Errorf("failed to read message: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"key":       string(message.Key),
		"topic":     message.Topic,
		"partition": message.Partition,
		"offset":    message.Offset,
	}).Debug("Successfully read message from Kafka")

	return message, nil
}

func (c *Consumer) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}
