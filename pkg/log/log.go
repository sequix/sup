package log

import (
	"fmt"
	"log"
	"os"
)

var g = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)

func Info(format string, args ...interface{}) {
	g.Output(2, fmt.Sprintf("INFO "+format, args...))
}

func Warn(format string, args ...interface{}) {
	g.Output(2, fmt.Sprintf("WARN "+format, args...))
}

func Error(format string, args ...interface{}) {
	g.Output(2, fmt.Sprintf("ERROR "+format, args...))
}

func Fatal(format string, args ...interface{}) {
	g.Output(2, fmt.Sprintf("FATAL "+format, args...))
	os.Exit(1)
}

func ErrorFunc(f func() error, format string, args ...interface{}) {
	if err := f(); err != nil {
		g.Output(2, fmt.Sprintf("ERROR "+format, args...)+fmt.Sprintf(" : %s", err))
	}
}
