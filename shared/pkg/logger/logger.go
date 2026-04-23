package logger

import (
	"log/slog"
	"os"
	"time"

	"shared/pkg/logger/slogpretty"
)

func New(env string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
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

	switch env {
	case "local":
		return newLocal(opts)
	case "dev":
		return newDev(opts)
	default:
		opts.Level = slog.LevelInfo
		return newProd(opts)
	}
}

func newLocal(opts *slog.HandlerOptions) *slog.Logger {
	prettyHandler := slogpretty.PrettyHandlerOptions{SlogOpts: opts}.
		NewPrettyHandler(os.Stdout)
	return slog.New(prettyHandler)
}

func newDev(opts *slog.HandlerOptions) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

func newProd(opts *slog.HandlerOptions) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, opts))
}
