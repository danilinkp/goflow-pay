package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"shared/pkg/logger/slogpretty"
)

func New(env string, level string, output string, file string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level:     logLevel,
		AddSource: true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				a.Value = slog.StringValue(
					t.Format(time.RFC3339),
				)
			}
			return a
		},
	}

	var writer io.Writer

	switch output {
	case "file":
		if file == "" {
			panic("log file path is required when output is file")
		}
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			panic("failed to create log dir: " + err.Error())
		}
		f, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			panic("failed to open log file: " + err.Error())
		}
		writer = f
	default:
		writer = os.Stdout
	}

	switch env {
	case "local":
		return newLocal(opts, writer)
	case "dev":
		return newDev(opts, writer)
	default:
		return newProd(opts, writer)
	}
}

func newLocal(opts *slog.HandlerOptions, writer io.Writer) *slog.Logger {
	prettyHandler := slogpretty.PrettyHandlerOptions{SlogOpts: opts}.
		NewPrettyHandler(writer)
	return slog.New(prettyHandler)
}

func newDev(opts *slog.HandlerOptions, writer io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(writer, opts))
}

func newProd(opts *slog.HandlerOptions, writer io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(writer, opts))
}
