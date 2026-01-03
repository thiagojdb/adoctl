package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	log = zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger()
}

func GetLogger() zerolog.Logger {
	return log
}

func SetLevel(level string) {
	var zerologLevel zerolog.Level
	switch level {
	case "debug":
		zerologLevel = zerolog.DebugLevel
	case "info":
		zerologLevel = zerolog.InfoLevel
	case "warn", "warning":
		zerologLevel = zerolog.WarnLevel
	case "error":
		zerologLevel = zerolog.ErrorLevel
	case "fatal":
		zerologLevel = zerolog.FatalLevel
	case "panic":
		zerologLevel = zerolog.PanicLevel
	default:
		zerologLevel = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(zerologLevel)
}

func Debug() *zerolog.Event {
	return log.Debug()
}

func Info() *zerolog.Event {
	return log.Info()
}

func Warn() *zerolog.Event {
	return log.Warn()
}

func Error() *zerolog.Event {
	return log.Error()
}

func Fatal() *zerolog.Event {
	return log.Fatal()
}

func Panic() *zerolog.Event {
	return log.Panic()
}

func Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}
