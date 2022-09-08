package opamp

import "log"

type Logger struct {
	Logger *log.Logger
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Logger.Printf(format, v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Logger.Printf(format, v...)
}

type NoopLogger struct{}

func (l *NoopLogger) Debugf(msg string, args ...interface{}) {}
func (l *NoopLogger) Errorf(msg string, args ...interface{}) {}
