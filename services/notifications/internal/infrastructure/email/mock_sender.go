package email

import (
	"context"
	"log/slog"
)

type MockSender struct {
	logger *slog.Logger
}

func NewMockSender(logger *slog.Logger) *MockSender {
	return &MockSender{
		logger: logger,
	}
}

func (m *MockSender) Send(_ context.Context, email string, title string, message string) error {
	m.logger.Info("sending email",
		slog.String("to", email),
		slog.String("title", title),
		slog.String("message", message),
	)
	return nil
}
