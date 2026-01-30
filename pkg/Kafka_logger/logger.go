package kafkalogger

import (
	"fmt"
	"os"
	"path/filepath"

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

type Logger_For_Tests struct {
}

func Create_Logger_For_Tests() *Logger_For_Tests {
	return &Logger_For_Tests{}
}

func (z *Logger_For_Tests) INFO(module, event, message string, userID *int) {

}

func (z *Logger_For_Tests) ERROR(module, event, message string, userID *int) {

}

func (z *Logger_For_Tests) DEBUG(module, event, message string, userID *int) {

}
func (z *Logger_For_Tests) WARN(module, event, message string, userID *int) {

}
