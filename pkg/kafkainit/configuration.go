// pkg/kafka/producer.go
package kafkainit

import (
	"github.com/segmentio/kafka-go"
)

const (
	Broker = "localhost:9092"
	Topic  = "my-topic"
)

func Set_Config() (error, *kafka.Conn) {

	conn, err := kafka.Dial("tcp", Broker)
	if err != nil {
		return err, nil
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(Topic)
	if err != nil || len(partitions) == 0 {

		topicConfig := kafka.TopicConfig{
			Topic:             Topic,
			NumPartitions:     3, // количество партиций
			ReplicationFactor: 1,
		}
		if err := conn.CreateTopics(topicConfig); err != nil {
			return err, nil
		}

	}
	return nil, conn
}
