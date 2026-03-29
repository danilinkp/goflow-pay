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
	username string
	password string
	from     string
}

func NewSmtpSender(host string, port int, username, password, from string) *SmtpSender {
	return &SmtpSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SmtpSender) Send(ctx context.Context, email string, title string, message string) error {
	from := "rudasocialnet@gmail.com"
	to := []string{email}

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	password := "unsldsybczcdmgqt"

	msg := fmt.Sprintf(
		"From: %s\r\n"+
			"To: %s\r\n"+
			"Subject: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=\"utf-8\"\r\n"+
			"\r\n"+
			"%s",
		from, email, title, message,
	)

	auth := smtp.PlainAuth("", from, password, smtpHost)

	addr := net.JoinHostPort(smtpHost, smtpPort)
	err := smtp.SendMail(addr, auth, from, to, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email to %s: %w", email, err)
	}

	return nil
}
