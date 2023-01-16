package logger

import (
	"log"
	"os"
	"strings"
)

type Logger interface {
	Errorf(format string, v ...interface{})
	Warnf(format string, v ...interface{})
	Debugf(format string, v ...interface{})
}

func CreateLogger() *logger {
	l := &logger{l: log.New(os.Stderr, "", log.Ldate|log.Lmicroseconds)}
	return l
}

var _ Logger = (*logger)(nil)

type logger struct {
	l *log.Logger
}

func (l *logger) Errorf(format string, v ...interface{}) {
	if strings.HasSuffix(format, ", Attempt %v") {
		return
	}
	l.output("ERROR RESTY "+format, v...)
}

func (l *logger) Warnf(format string, v ...interface{}) {
	l.output("WARN RESTY "+format, v...)
}

func (l *logger) Debugf(format string, v ...interface{}) {
	l.output("DEBUG RESTY "+format, v...)
}

func (l *logger) output(format string, v ...interface{}) {
	if len(v) == 0 {
		l.l.Print(format)
		return
	}
	l.l.Printf(format, v...)
}
