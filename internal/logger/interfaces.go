package logger

import "io"

type Logger interface {
	Info(message string, v ...interface{})
	Warn(message string, v ...interface{})
	Error(message string, v ...interface{})
	Debug(message string, v ...interface{})
	GetWriter() io.Writer
}
