package obs

import (
	"log"
	"os"
)

// Logger provides minimal structured logging. Can be replaced with slog or zerolog later.
type Logger struct {
	*log.Logger
}

// NewLogger returns a logger writing to stderr.
func NewLogger() *Logger {
	return &Logger{Logger: log.New(os.Stderr, "", log.LstdFlags)}
}
