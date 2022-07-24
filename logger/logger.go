package logger

import (
	"fmt"
	"log"
	"os"
)

type logger struct {
	prefix      string
	innerLogger *log.Logger
}

func GetLogger(prefix string) Logger {
	return &logger{
		prefix:      prefix,
		innerLogger: log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds),
	}
}

func (l *logger) Log(message string, v ...interface{}) {
	l.innerLogger.Printf("%v %v\n", l.prefix, fmt.Sprintf(message, v...))
}
