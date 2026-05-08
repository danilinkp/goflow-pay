package sl

import (
	"fmt"
	"log/slog"
	"runtime"
	"time"

	"github.com/google/uuid"
)

func Op(op string) slog.Attr {
	return slog.String("op", op)
}

func EventID() slog.Attr {
	return slog.String("event_id", uuid.New().String())
}

func Duration(d time.Duration) slog.Attr {
	return slog.String("logic_time", d.String())
}

func Err(err error) slog.Attr {
	return slog.Group("error",
		slog.String("message", err.Error()),
		slog.String("type", fmt.Sprintf("%T", err)),
	)
}

func ErrWithStack(err error) slog.Attr {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return slog.Group("error",
		slog.String("message", err.Error()),
		slog.String("type", fmt.Sprintf("%T", err)),
		slog.String("stack", string(buf[:n])),
	)
}
