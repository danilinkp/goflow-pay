package email

import (
	"context"
	"log/slog"
)

type MockSender struct {
	logger *slog.Logger
}

func (m *MockSender) Send(ctx context.Context, email string, title string, message string) error {
	m.logger.Info("sending email",
		slog.String("to", email),
		slog.String("title", title),
		slog.String("message", message),
	)
	return nil
}
