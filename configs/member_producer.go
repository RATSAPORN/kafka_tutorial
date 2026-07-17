package configs

import (
	"encoding/json"
	"log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/username/myproject/dtos"
)

const MemberCreatedTopic = "member.created"

type MemberProducer interface {
	PublishMemberCreated(event *dtos.MemberEvent) error
}

type memberProducer struct {
	producer *kafka.Producer
}

func NewMemberProducer(p *kafka.Producer) MemberProducer {
	return &memberProducer{producer: p}
}

func (m *memberProducer) PublishMemberCreated(event *dtos.MemberEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	topic := MemberCreatedTopic
	err = m.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: payload,
	}, nil)

	if err != nil {
		return err
	}

	log.Printf("Published event to topic %s", topic)
	return nil
}
