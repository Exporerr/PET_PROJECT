package kafkalogger

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

type Logger struct {
	Service  string
	logChan  chan Logger_Message
	stop     chan struct{}
	Buffer   []Logger_Message
	max_size int
	file     *os.File
	wg       *sync.WaitGroup
	mu       *sync.Mutex
	timer    time.Ticker
}
type Logger_Message struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Service   string    `json:"service"` // имя микросервиса
	Module    string    `json:"module"`  // handler, service, repository
	Event     string    `json:"event"`
	Message   string    `json:"message"`
	User_ID   *int      `json:"user_id,omitempty"` // необязательно
}

func New_Logger(file *os.File, max_size int, service string, wg *sync.WaitGroup, mu *sync.Mutex, tim time.Duration) *Logger {
	l := &Logger{
		logChan:  make(chan Logger_Message, max_size),
		stop:     make(chan struct{}),
		Buffer:   []Logger_Message{},
		max_size: max_size,
		file:     file,
		wg:       wg,
		mu:       mu,
		timer:    *time.NewTicker(tim),
	}
	l.wg.Add(1)
	go l.run()
	return l
}
func (l *Logger) run() {
	defer l.wg.Done()
	for {
		select {
		case msg := <-l.logChan:
			l.Buffer = append(l.Buffer, msg)
			if len(l.Buffer) >= l.max_size {
				l.flush()
			}
		case <-l.timer.C:
			if len(l.Buffer) > 0 {
				l.flush()
			}
		case <-l.stop:
			if len(l.Buffer) > 0 {
				l.flush()
			}
			return

		}
	}
}
func (l *Logger) flush() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, msg := range l.Buffer {
		json_msg, _ := json.Marshal(msg)
		_, _ = l.file.Write(append(json_msg, '\n'))
	}
	l.Buffer = l.Buffer[:0]
}
func (log *Logger) Create_log(module string, level string, event string, message string, user_id *int) {

	log_message := &Logger_Message{
		Level:     level,
		Service:   log.Service,
		Module:    module,
		Event:     event,
		Message:   message,
		User_ID:   user_id,
		Timestamp: time.Now(),
	}
	select {
	case log.logChan <- *log_message:
	default:
		fmt.Fprintf(os.Stderr, "log channel full — dropped message: %s\n", message)
	}

}

func (l *Logger) INFO(module string, event string, message string, user_id *int) {
	l.Create_log(module, "INFO", event, message, user_id)
}
func (l *Logger) ERROR(module string, event string, message string, user_id *int) {
	l.Create_log(module, "ERROR", event, message, user_id)
}
func (l *Logger) DEBUG(module string, event string, message string, user_id *int) {
	l.Create_log(module, "DEBUG", event, message, user_id)
}
func (l *Logger) Close() {
	l.timer.Stop()
	close(l.stop)
	l.wg.Wait()

	l.file.Close()
}
func NewNop() *Logger {
	return &Logger{
		// инициализируй безопасные пустые поля
		// или просто флаги "disabled"
	}
}
