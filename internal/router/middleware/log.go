package middleware

import (
	"log/slog"
	"os"
)

func SetopLogger() *slog.Logger {
	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	return log
}
