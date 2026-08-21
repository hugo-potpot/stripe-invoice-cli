package mailer

import (
	"context"
	"fmt"
	"log/slog"
	"stripe-invoice-go/internal/domain"

	"github.com/wneessen/go-mail"
)

type SMTPMailer struct {
	host     string
	port     int
	username string
	password string
	from     string
	to       string
}

func NewSMTPMailer(host string, port int, username, password, from, to string) *SMTPMailer {
	return &SMTPMailer{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
		to:       to,
	}
}

func (m *SMTPMailer) Send(ctx context.Context, period domain.Period, attachments []string) error {
	message := mail.NewMsg()
	if err := message.From(m.from); err != nil {
		return fmt.Errorf("setting from address: %w", err)
	}
	if err := message.To(m.to); err != nil {
		return fmt.Errorf("setting to address: %w", err)
	}

	subject := createSubject(period)
	message.Subject(subject)
	message.SetBodyString(mail.TypeTextPlain, createBody(period))

	for _, attachment := range attachments {
		message.AttachFile(attachment)
	}

	client, err := mail.NewClient(m.host, mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(m.username), mail.WithPassword(m.password), mail.WithPort(m.port))
	if err != nil {
		return err
	}

	slog.InfoContext(ctx, "sending mail", "to", m.to, "subject", subject, "attachments", len(attachments))
	if err := client.DialAndSend(message); err != nil {
		return fmt.Errorf("sending mail: %w", err)
	}

	slog.InfoContext(ctx, "mail sent", "to", m.to)
	return nil
}

func createSubject(period domain.Period) string {
	return fmt.Sprintf("Stripe Invoice for %s", period.String())
}
func createBody(period domain.Period) string {
	return fmt.Sprintf("Hi,\n\nPlease find attached the Stripe documents for the month of %s.\n\nCordially.", period.String())
}
