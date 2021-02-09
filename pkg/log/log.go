package log

import (
	"fmt"
	"log"
	"os"
)

var g *log.Logger

func Init() {
	g = log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile|log.LUTC)
}

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
