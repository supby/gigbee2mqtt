package logger

type Logger interface {
	Log(message string, v ...interface{})
}
