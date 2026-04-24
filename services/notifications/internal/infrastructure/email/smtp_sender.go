package email

import (
	"context"
	"fmt"
	"net"
	"net/smtp"
)

type SmtpSender struct {
	host     string
	port     int
	password string
	from     string
}

func NewSmtpSender(host string, port int, password, from string) *SmtpSender {
	return &SmtpSender{
		host:     host,
		port:     port,
		password: password,
		from:     from,
	}
}

func (s *SmtpSender) Send(_ context.Context, email string, title string, message string) error {
	to := []string{email}

	msg := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=\"utf-8\"\r\n"+
			"\r\n"+
			"%s",
		s.from, email, title, message,
	)

	auth := smtp.PlainAuth("", s.from, s.password, s.host)

	addr := net.JoinHostPort(s.host, string(rune(s.port)))
	err := smtp.SendMail(addr, auth, s.from, to, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email to %s: %w", email, err)
	}

	return nil
}
