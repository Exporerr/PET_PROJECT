package kafkainit

import (
	"context"

	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	Consum *kafka.Reader
	Log    *kafkalogger.ZapAdapter
	Wg     sync.WaitGroup

	File *os.File
}

const (
	Broker = "kafka:9092"
	Topic  = "my-topic"
)

func New_Consumer(log *kafkalogger.ZapAdapter, ctx context.Context) (*Consumer, error, func() error) {
	eventsBase := os.Getenv("CONS_EVENTS")
	if eventsBase == "" {
		eventsBase = "USER-EVENTS"
	}
	if err := os.MkdirAll(eventsBase, 0755); err != nil {
		return nil, fmt.Errorf("mkdir log_event folder %v", err), nil
	}
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.000000")
	logfilepath := filepath.Join(eventsBase, fmt.Sprintf("%s.log", timestamp))
	logfile, err := os.OpenFile(logfilepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.ERROR("Consumer ", "File opening", fmt.Sprintf("failed to open file :%v", err), nil)
		return nil, err, nil
	}
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{Broker},
		Topic:       Topic,
		MaxAttempts: 4,
		GroupID:     "my-groupID",
	})
	cons := Consumer{
		Consum: reader,
		Log:    log,
		Wg:     sync.WaitGroup{},

		File: logfile,
	}
	cons.Wg.Add(1)

	return &cons, nil, logfile.Close

}

func (c *Consumer) Run(ctx context.Context) {
	defer c.Wg.Done()

	for {
		msg, err := c.Consum.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.Log.ERROR("Consumer", "ReadMessage", err.Error(), nil)
			continue
		}

		c.process(msg)
	}
}

func (c *Consumer) process(msg kafka.Message) {

	var user_action models.User_Action
	if erro := json.Unmarshal(msg.Value, &user_action); erro != nil {
		c.Log.ERROR("process", "unmarshaling error", erro.Error(), nil)
	}

	out, erro := json.Marshal(user_action)
	if erro != nil {
		c.Log.ERROR("process", "marshaling error", erro.Error(), nil)
	}

	_, err := c.File.Write(append(out, '\n'))
	if err != nil {
		c.Log.ERROR("process", "writing to file failed", err.Error(), nil)
		return
	}

}

func (c *Consumer) Close() {
	c.Wg.Wait()
	if err := c.Consum.Close(); err != nil {
		c.Log.ERROR("Kafka Consumer", "Close", err.Error(), nil)
	}

}
