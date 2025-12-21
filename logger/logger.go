package logger

import (
	"log/slog"
	"os"
)

var Log *slog.Logger

func InitLogger() {
	config := slog.HandlerOptions{
		AddSource: true,
	}
	handler := slog.NewTextHandler(os.Stderr, &config)
	Log = slog.New(handler)

}

func Error(msg string, err error) {
	Log.Error(
		"error start application",
		"error", err,
	)
}
