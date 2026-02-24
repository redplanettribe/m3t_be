package domain

import "context"

// Mailer defines the contract for sending emails (infrastructure port).
type Mailer interface {
	Send(to, subject, html, text string) error
}

// EmailTemplateRenderer renders email content from a named template with the given data.
type EmailTemplateRenderer interface {
	Render(templateName string, data any) (subject, htmlBody, textBody string, err error)
}

// WelcomeMessageEmailData holds data for the welcome email.
type WelcomeMessageEmailData struct {
	Email     string
	FirstName string
	UserID    string // optional, for future use
	Language  string // optional, for future locale/templates
}

// LoginCodeEmailData holds data for the passwordless login code email.
type LoginCodeEmailData struct {
	Email             string
	Code              string
	ExpiresInMinutes  int
}

// EmailService defines the contract for sending domain-level emails.
type EmailService interface {
	SendWelcomeMessage(ctx context.Context, data *WelcomeMessageEmailData) error
	SendLoginCode(ctx context.Context, data *LoginCodeEmailData) error
}
