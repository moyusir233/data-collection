package logger

import (
	"fmt"
	v1 "gitee.com/moyusir/util/api/util/v1"
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

func NewJsonZapLoggerWarpper(serviceName string, level v1.LogLevel) *Logger {
	encoderConfig := ecszap.NewDefaultEncoderConfig()
	// 由于protobuf枚举的最小值为0，而zap logger level的最小值为-1，因此需要减一进行偏移
	core := ecszap.NewCore(encoderConfig, os.Stdout, zapcore.Level(level-1))
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
