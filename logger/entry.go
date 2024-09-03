package logger

import (
	"errors"
	"sync"

	"go.uber.org/zap"
)

var locker sync.RWMutex

func Debug(msg string, field ...zap.Field) {
	locker.RLock()
	defer locker.RUnlock()
	defaultLogger.Debug(msg, field...)
}

func Info(msg string, field ...zap.Field) {
	locker.RLock()
	defer locker.RUnlock()
	defaultLogger.Info(msg, field...)
}

func Warn(msg string, field ...zap.Field) {
	locker.RLock()
	defer locker.RUnlock()
	defaultLogger.Warn(msg, field...)
}

func Error(msg string, field ...zap.Field) {
	locker.RLock()
	defer locker.RUnlock()
	defaultLogger.Error(msg, field...)
}

func Fatal(msg string, field ...zap.Field) {
	locker.RLock()
	defer locker.RUnlock()
	defaultLogger.Fatal(msg, field...)
}

func Default() *zap.Logger {
	locker.RLock()
	defer locker.RUnlock()
	return defaultLogger
}

func L(moduleName string) (*zap.Logger, error) {
	locker.RLock()
	defer locker.RUnlock()
	logger := loggers[moduleName+"-logger"]
	if logger == nil {
		return nil, errors.New("logger not found")
	}
	return logger, nil
}
