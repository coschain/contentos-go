package mylog

import (
	"os"

	"github.com/sirupsen/logrus"
)

// const
const (
	PanicLevel = "panic"
	FatalLevel = "fatal"
	ErrorLevel = "error"
	WarnLevel  = "warn"
	InfoLevel  = "info"
	DebugLevel = "debug"
)

type MyLog struct {
	Logger *logrus.Logger
}

type emptyWriter struct{}

func (ew emptyWriter) Write(p []byte) (int, error) {
	return 0, nil
}

func convertLevel(level string) logrus.Level {
	switch level {
	case PanicLevel:
		return logrus.PanicLevel
	case FatalLevel:
		return logrus.FatalLevel
	case ErrorLevel:
		return logrus.ErrorLevel
	case WarnLevel:
		return logrus.WarnLevel
	case InfoLevel:
		return logrus.InfoLevel
	case DebugLevel:
		return logrus.DebugLevel
	default:
		return logrus.InfoLevel
	}
}

func NewMyLog(path string, level string, age uint32) (*MyLog, error){
	mylog := &MyLog{}
	mylog.Logger = Init(path, level, age)
	return mylog, nil
}

// Init loggers
func Init(path string, level string, age uint32) *logrus.Logger {
	fileHooker := NewFileRotateHooker(path, age)
	var clog *logrus.Logger

	clog = logrus.New()
	LoadFunctionHooker(clog)
	clog.Hooks.Add(fileHooker)
	clog.Out = os.Stdout
	clog.Formatter = &TextFormatter{
		ForceColors:     true,
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
		ForceFormatting: true,
	}
	clog.Level = convertLevel(level)

	return clog
}