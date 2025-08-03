package kafka

import (
	"context"
	"fmt"
	"time"

	"todo/internal/config"

	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

// Client wraps the Kafka client with additional functionality
type Client struct {
	producer sarama.SyncProducer
	consumer sarama.Consumer
	config   *config.KafkaConfig
	logger   *zap.Logger
}

// NewClient creates a new Kafka client
func NewClient(config *config.Config, logger *zap.Logger) (*Client, error) {
	if !config.Kafka.Enabled {
		logger.Info("Kafka is disabled, skipping connection")
		return &Client{
			producer: nil,
			consumer: nil,
			config:   &config.Kafka,
			logger:   logger,
		}, nil
	}

	// Configure producer
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.RequiredAcks = sarama.WaitForAll
	producerConfig.Producer.Retry.Max = 3
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.Timeout = 5 * time.Second

	// Create producer
	producer, err := sarama.NewSyncProducer(config.Kafka.Brokers, producerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
	}

	// Configure consumer
	consumerConfig := sarama.NewConfig()
	consumerConfig.Consumer.Return.Errors = true
	consumerConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	// Create consumer
	consumer, err := sarama.NewConsumer(config.Kafka.Brokers, consumerConfig)
	if err != nil {
		producer.Close()
		return nil, fmt.Errorf("failed to create Kafka consumer: %w", err)
	}

	logger.Info("Kafka connected successfully",
		zap.Strings("brokers", config.Kafka.Brokers),
		zap.String("topic", config.Kafka.Topic),
	)

	return &Client{
		producer: producer,
		consumer: consumer,
		config:   &config.Kafka,
		logger:   logger,
	}, nil
}

// IsEnabled returns whether Kafka is enabled
func (c *Client) IsEnabled() bool {
	return c.config.Enabled && c.producer != nil && c.consumer != nil
}

// SendMessage sends a message to Kafka
func (c *Client) SendMessage(ctx context.Context, key, value string) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Kafka is not enabled")
	}

	msg := &sarama.ProducerMessage{
		Topic: c.config.Topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.StringEncoder(value),
	}

	partition, offset, err := c.producer.SendMessage(msg)
	if err != nil {
		return fmt.Errorf("failed to send message to Kafka: %w", err)
	}

	c.logger.Debug("Message sent to Kafka",
		zap.String("topic", c.config.Topic),
		zap.Int32("partition", partition),
		zap.Int64("offset", offset),
		zap.String("key", key),
	)

	return nil
}

// ConsumeMessages starts consuming messages from Kafka
func (c *Client) ConsumeMessages(ctx context.Context, handler func(key, value string) error) error {
	if !c.IsEnabled() {
		return fmt.Errorf("Kafka is not enabled")
	}

	partitionConsumer, err := c.consumer.ConsumePartition(c.config.Topic, 0, sarama.OffsetNewest)
	if err != nil {
		return fmt.Errorf("failed to create partition consumer: %w", err)
	}
	defer partitionConsumer.Close()

	c.logger.Info("Started consuming messages from Kafka",
		zap.String("topic", c.config.Topic),
	)

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("Stopping Kafka consumer due to context cancellation")
			return nil
		case msg := <-partitionConsumer.Messages():
			key := string(msg.Key)
			value := string(msg.Value)

			c.logger.Debug("Received message from Kafka",
				zap.String("topic", msg.Topic),
				zap.Int32("partition", msg.Partition),
				zap.Int64("offset", msg.Offset),
				zap.String("key", key),
			)

			if err := handler(key, value); err != nil {
				c.logger.Error("Error handling Kafka message", zap.Error(err))
			}
		case err := <-partitionConsumer.Errors():
			c.logger.Error("Error consuming from Kafka", zap.Error(err))
		}
	}
}

// Close gracefully closes the Kafka connections
func (c *Client) Close() error {
	var errors []error

	if c.producer != nil {
		c.logger.Info("Closing Kafka producer...")
		if err := c.producer.Close(); err != nil {
			c.logger.Error("Error closing Kafka producer", zap.Error(err))
			errors = append(errors, err)
		} else {
			c.logger.Info("Kafka producer closed successfully")
		}
	}

	if c.consumer != nil {
		c.logger.Info("Closing Kafka consumer...")
		if err := c.consumer.Close(); err != nil {
			c.logger.Error("Error closing Kafka consumer", zap.Error(err))
			errors = append(errors, err)
		} else {
			c.logger.Info("Kafka consumer closed successfully")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors closing Kafka connections: %v", errors)
	}

	return nil
}
