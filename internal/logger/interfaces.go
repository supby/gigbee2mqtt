package logger

import "io"

type Logger interface {
	Log(message string, v ...interface{})
	GetWriter() io.Writer
}
