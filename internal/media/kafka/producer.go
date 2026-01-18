package kafka

import (
	"context"
	"fmt"

	kafkago "github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	return &Producer{
		writer: &kafkago.Writer{
			Addr:     kafkago.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafkago.LeastBytes{},
		},
	}
}

func (p *Producer) Publish(ctx context.Context, key string, value []byte) error {
	err := p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(key),
		Value: value,
	})
	if err != nil {
		return fmt.Errorf("kafka publish: %w", err)
	}
	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
