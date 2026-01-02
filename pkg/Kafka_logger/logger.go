package kafkalogger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LoggerInterface interface {
	INFO(module, event, message string, userID *int)
	ERROR(module, event, message string, userID *int)
	DEBUG(module, event, message string, userID *int)
}

type ZapAdapter struct {
	Logger  *zap.Logger
	Service string
}

func New_ZapAdapter(service string, loglevel string) (*ZapAdapter, func() error, error) {
	lvl := zap.NewAtomicLevel()
	if err := lvl.UnmarshalText([]byte(loglevel)); err != nil {
		return nil, nil, fmt.Errorf("unmarshal log level %v", err)
	}
	if err := os.MkdirAll("logs", 0755); err != nil {
		return nil, nil, fmt.Errorf("mkdir log folder %v", err)
	}
	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.000000")
	logfilepath := filepath.Join("logs", fmt.Sprintf("%s.log", timestamp))
	logfile, err := os.OpenFile(logfilepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file: %v", err)
	}
	cfg := zap.NewProductionEncoderConfig()

	cfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05.000000")
	encoder := zapcore.NewConsoleEncoder(cfg)
	enc := zapcore.NewJSONEncoder(cfg)

	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), lvl),
		zapcore.NewCore(enc, zapcore.AddSync(logfile), lvl),
	)
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return &ZapAdapter{
		Logger:  logger,
		Service: service,
	}, logfile.Close, nil

}
func (z *ZapAdapter) buildFields(module, event string, userID *int) []zap.Field {
	fields := []zap.Field{
		zap.String("service", z.Service),
		zap.String("module", module),
		zap.String("event", event),
	}
	if userID != nil {
		fields = append(fields, zap.Int("user_id", *userID))
	}
	return fields
}

func (z *ZapAdapter) INFO(module, event, message string, userID *int) {
	z.Logger.Info(message, z.buildFields(module, event, userID)...)
}

func (z *ZapAdapter) ERROR(module, event, message string, userID *int) {
	z.Logger.Error(message, z.buildFields(module, event, userID)...)
}

func (z *ZapAdapter) DEBUG(module, event, message string, userID *int) {
	z.Logger.Debug(message, z.buildFields(module, event, userID)...)
}
func (z *ZapAdapter) WARN(module, event, message string, userID *int) {
	z.Logger.Warn(message, z.buildFields(module, event, userID)...)
}

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
