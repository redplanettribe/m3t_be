package services

import (
	"context"
	"fmt"
	"log"

	"multitrackticketing/internal/domain"
)

type emailService struct {
	mailer   domain.Mailer
	renderer domain.EmailTemplateRenderer
}

// NewEmailService returns an EmailService that uses the given Mailer and template renderer.
func NewEmailService(mailer domain.Mailer, renderer domain.EmailTemplateRenderer) domain.EmailService {
	return &emailService{mailer: mailer, renderer: renderer}
}

// SendWelcomeMessage sends a welcome email using the "welcome" template and the given data.
func (s *emailService) SendWelcomeMessage(ctx context.Context, data *domain.WelcomeMessageEmailData) error {
	if data == nil {
		return fmt.Errorf("welcome message data is nil")
	}
	subject, htmlBody, textBody, err := s.renderer.Render("welcome", data)
	if err != nil {
		return fmt.Errorf("failed to render welcome template: %w", err)
	}
	if err := s.mailer.Send(data.Email, subject, htmlBody, textBody); err != nil {
		return fmt.Errorf("failed to send welcome email: %w", err)
	}
	log.Printf("[EMAIL] Welcome email sent to %s", data.Email)
	return nil
}

// SendLoginCode sends the passwordless login code email using the "login_code" template.
func (s *emailService) SendLoginCode(ctx context.Context, data *domain.LoginCodeEmailData) error {
	if data == nil {
		return fmt.Errorf("login code email data is nil")
	}
	subject, htmlBody, textBody, err := s.renderer.Render("login_code", data)
	if err != nil {
		return fmt.Errorf("failed to render login_code template: %w", err)
	}
	if err := s.mailer.Send(data.Email, subject, htmlBody, textBody); err != nil {
		return fmt.Errorf("failed to send login code email: %w", err)
	}
	log.Printf("[EMAIL] Login code sent to %s", data.Email)
	return nil
}
