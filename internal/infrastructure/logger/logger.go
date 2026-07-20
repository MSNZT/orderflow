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

func Op(op string) slog.Attr {
	return slog.Attr{
		Key:   "op",
		Value: slog.StringValue(op),
	}
}

func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "err",
		Value: slog.StringValue(err.Error()),
	}
}
