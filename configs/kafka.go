package configs

import (
	"context"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func NewKafkaProducer(broker string) *kafka.Producer {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": broker,
	})
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	return p
}

func CreateKafkaTopics(broker string) {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{
		"bootstrap.servers": broker,
	})
	if err != nil {
		log.Fatalf("Failed to create Kafka admin client: %v", err)
	}
	defer adminClient.Close()

	topics := []kafka.TopicSpecification{
		{
			Topic:             MemberCreatedTopic,
			NumPartitions:     1,
			ReplicationFactor: 1,
		},
	}

	results, err := adminClient.CreateTopics(context.Background(), topics)
	if err != nil {
		log.Fatalf("Failed to create topics: %v", err)
	}

	for _, result := range results {
		if result.Error.Code() != kafka.ErrNoError &&
			result.Error.Code() != kafka.ErrTopicAlreadyExists {
			log.Fatalf("Failed to create topic %s: %v", result.Topic, result.Error)
		}
		log.Printf("Topic ready: %s", result.Topic)
	}
}
