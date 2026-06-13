package mailer

import "fmt"

// Config holds the settings needed to construct a Mailer.
type Config struct {
	Provider     string // "resend" or "smtp"
	FromEmail    string
	ResendAPIKey string
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
}

// New constructs a Mailer based on cfg.Provider.
func New(cfg Config) (Mailer, error) {
	switch cfg.Provider {
	case "resend":
		return NewResendMailer(cfg.ResendAPIKey, cfg.FromEmail), nil
	case "smtp":
		return NewSMTPMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.FromEmail), nil
	default:
		return nil, fmt.Errorf("unknown email provider %q", cfg.Provider)
	}
}
