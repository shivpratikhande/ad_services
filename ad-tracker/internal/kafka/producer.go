package kafka

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

func NewKafkaWriter(brokerURL, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:     kafka.TCP(brokerURL),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}
}

func PublishEvent(writer *kafka.Writer, key, value []byte) {
	err := writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   key,
			Value: value,
		},
	)
	if err != nil {
		log.Printf("Failed to write message to Kafka: %v", err)
	}
}
