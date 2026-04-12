package kafka

import (
	"log"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/Kash4299/todo-chat-app/internal/config"
)

type KafkaClient struct {
	Producer sarama.SyncProducer
	Consumer sarama.Consumer
}

func NewKafkaClient(cfg *config.Config) *KafkaClient {
	brokers := strings.Split(cfg.KafkaBrokers, ",")

	// Producer
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.Timeout = 5 * time.Second
	producerConfig.Net.DialTimeout = 5 * time.Second

	producer, err := sarama.NewSyncProducer(brokers, producerConfig)
	if err != nil {
		log.Printf("WARNING: failed to create Kafka producer: %v", err)
	} else {
		log.Println("Kafka producer connected successfully")
	}

	// Consumer
	consumerConfig := sarama.NewConfig()
	consumerConfig.Net.DialTimeout = 5 * time.Second

	consumer, err := sarama.NewConsumer(brokers, consumerConfig)
	if err != nil {
		log.Printf("WARNING: failed to create Kafka consumer: %v", err)
	} else {
		log.Println("Kafka consumer connected successfully")
	}

	return &KafkaClient{
		Producer: producer,
		Consumer: consumer,
	}
}

func (k *KafkaClient) Close() {
	if k.Producer != nil {
		if err := k.Producer.Close(); err != nil {
			log.Printf("error closing Kafka producer: %v", err)
		}
	}
	if k.Consumer != nil {
		if err := k.Consumer.Close(); err != nil {
			log.Printf("error closing Kafka consumer: %v", err)
		}
	}
}
