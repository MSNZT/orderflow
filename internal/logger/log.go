package logger

import (
	"log/slog"
	"os"
)

func New() *slog.Logger {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	return log
}
