package kafkainit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"strconv"
	"sync"
	"time"

	kafkalogger "github.com/Explorerr/pet_project/pkg/Kafka_logger"
	models "github.com/Explorerr/pet_project/pkg/Models"
	contextkeys "github.com/Explorerr/pet_project/pkg/context_Key"
	"github.com/segmentio/kafka-go"
)

type Producer interface {
	WriteMessage()
}

type Producer_real struct {
	Writer *kafka.Writer

	Message_chan chan kafka.Message
	Buffer       []kafka.Message
	Max_size     int
	Wg           *sync.WaitGroup
	Mu           *sync.Mutex
	Timer        time.Ticker
	Log          *kafkalogger.ZapAdapter
}

func New_Producer(log *kafkalogger.ZapAdapter, max_size int, tim time.Duration) *Producer_real {
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:      []string{Broker},
		Topic:        Topic,
		RequiredAcks: -1,
		MaxAttempts:  4,
		BatchSize:    10,
		WriteTimeout: 4 * time.Second,
		Balancer:     &kafka.Hash{},
	})

	producer := &Producer_real{
		Max_size:     max_size,
		Wg:           &sync.WaitGroup{},
		Mu:           &sync.Mutex{},
		Timer:        *time.NewTicker(tim),
		Buffer:       []kafka.Message{},
		Message_chan: make(chan kafka.Message, max_size),
		Writer:       writer,
		Log:          log,
	}

	return producer

}

func (p *Producer_real) WriteMessagee(userID int, action string, resourceID int, r *http.Request) {
	rd := r.Context().Value(contextkeys.ReqInfo).(*models.Request_Info)
	userAction := models.User_Action{
		User_id:   userID,
		Action:    action,
		Resource:  resourceID,
		IP:        rd.Ip_add,
		Method:    rd.Method,
		Path:      rd.Path,
		UserAgent: rd.Source,
		Timestamp: time.Now(),
	}
	data, err := json.Marshal(userAction)
	if err != nil {
		p.Log.ERROR("Kafka Producer", "JSON Marshaling", "failed to marshal msg", &userID)
		return
	}

	msg := &kafka.Message{
		Key:   []byte(strconv.Itoa(userID)),
		Value: []byte(data),
	}
	select {
	case p.Message_chan <- *msg:
	default:

		p.Log.ERROR("Kafka Producer", "WriteMessage to channel", "log channel full — dropped message", &userID)

	}
}

func (p *Producer_real) Run(ctx context.Context) {
	defer p.Wg.Done()

	for {
		select {
		case <-p.Timer.C:
			if len(p.Buffer) > 0 {
				p.flush()
			}
		case <-ctx.Done():
			if len(p.Buffer) > 0 {
				p.flush()
			}
			return

		case msg := <-p.Message_chan:
			if len(p.Buffer) >= p.Max_size {
				p.flush()

			}
			p.Buffer = append(p.Buffer, msg)
		}
	}
}

func (p *Producer_real) flush() {
	p.Mu.Lock()
	defer p.Mu.Unlock()

	err := p.Writer.WriteMessages(context.Background(), p.Buffer...)
	if err != nil {
		p.Log.ERROR(
			"Kafka Producer",
			"Flush",
			fmt.Sprintf("kafka write error: %v", err),
			nil,
		)
	}

	p.Buffer = p.Buffer[:0]

}
func (p *Producer_real) Close() {
	p.Timer.Stop()

	p.Wg.Wait() // ждём завершения run()

	if err := p.Writer.Close(); err != nil {
		p.Log.ERROR("Kafka Producer", "Close", err.Error(), nil)
	}
}
