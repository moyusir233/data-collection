package logger

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var _ log.Logger = (*Logger)(nil)

type Logger struct {
	log *zap.Logger
}

func NewJsonZapLoggerWarpper(serviceName string) *Logger {
	encoderConfig := ecszap.NewDefaultEncoderConfig()
	core := ecszap.NewCore(encoderConfig, os.Stdout, zapcore.InfoLevel)
	zapLogger := zap.New(core).Named(serviceName)

	return &Logger{
		log: zapLogger,
	}
}

func (l *Logger) Log(level log.Level, keyvals ...interface{}) error {
	if len(keyvals) == 0 || len(keyvals)%2 != 0 {
		l.log.Warn(fmt.Sprint("Keyvalues must appear in pairs: ", keyvals))
		return nil
	}

	var data []zap.Field
	var msg string
	for i := 0; i < len(keyvals); i += 2 {
		// 处理logger在被helper封装后重复输出msg的问题
		if key, ok := keyvals[i].(string); ok && key == "msg" {
			msg = fmt.Sprint(keyvals[i+1])
			continue
		}
		data = append(data, zap.Any(fmt.Sprint(keyvals[i]), keyvals[i+1]))
	}

	switch level {
	case log.LevelDebug:
		l.log.Debug(msg, data...)
	case log.LevelInfo:
		l.log.Info(msg, data...)
	case log.LevelWarn:
		l.log.Warn(msg, data...)
	case log.LevelError:
		l.log.Error(msg, data...)
	case log.LevelFatal:
		l.log.Fatal(msg, data...)
	}
	return nil
}

func (l *Logger) Sync() error {
	return l.log.Sync()
}
