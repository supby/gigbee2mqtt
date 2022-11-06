package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

const (
	LogLevelInfo  = 0
	LogLevelWarn  = 1
	LogLevelError = 2
	LogLevelDebug = 3
)

type logger struct {
	prefix      string
	innerLogger *log.Logger
	level       int
}

func GetLogger(prefix string, level int) Logger {
	return &logger{
		prefix:      prefix,
		innerLogger: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds),
		level:       level,
	}
}

func (l *logger) Info(message string, v ...interface{}) {
	l.log(fmt.Sprintf("[INFO] %v", message), v...)
}

func (l *logger) Warn(message string, v ...interface{}) {
	if l.level < LogLevelWarn {
		return
	}

	l.log(fmt.Sprintf("[WARN] %v", message), v...)
}

func (l *logger) Error(message string, v ...interface{}) {
	if l.level < LogLevelError {
		return
	}

	l.log(fmt.Sprintf("[ERROR] %v", message), v...)
}

func (l *logger) Debug(message string, v ...interface{}) {
	if l.level < LogLevelDebug {
		return
	}

	l.log(fmt.Sprintf("[DEBUG] %v", message), v...)
}

func (l *logger) log(message string, v ...interface{}) {
	l.innerLogger.Printf("%v %v\n", l.prefix, fmt.Sprintf(message, v...))
}

func (l *logger) GetWriter() io.Writer {
	return l.innerLogger.Writer()
}
