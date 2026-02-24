---
name: add-email
description: Add a new email type or change the email implementation. Use when adding a new transactional email (e.g. password reset), new template, or when modifying how email is sent or rendered.
---

# Add Email

Follow project rules in `.cursor/rules/` (Clean Architecture, [email.mdc](.cursor/rules/email.mdc)).

## When to use

- Adding a new transactional email (e.g. welcome, password reset, notification)
- Adding or editing email templates
- Changing how the mailer or template renderer works

## Architecture summary

- **Domain** (`internal/domain/email.go`): `Mailer`, `EmailTemplateRenderer`, `EmailService` interfaces; email DTOs (e.g. `WelcomeMessageEmailData`).
- **Adapter** (`internal/adapters/email/`): Implements Mailer (SES/noop) and EmailTemplateRenderer (embedded templates from `templates/`).
- **Service** (`internal/services/email.go`): Implements `EmailService`; uses renderer then mailer. No inline HTML/text—always use templates.

## Adding a new email type (e.g. password reset)

1. **Domain**: In `internal/domain/email.go`, add a DTO struct for the email data (e.g. `PasswordResetEmailData` with `Email`, `ResetToken`, `ResetURL`). Add the method to `EmailService` (e.g. `SendPasswordResetEmail(ctx context.Context, data *PasswordResetEmailData) error`).

2. **Templates**: Under `internal/adapters/email/templates/`, add three files for template name `password_reset`:
   - `password_reset_subject.txt` (e.g. `Password reset request`)
   - `password_reset.html` (HTML body; use `{{.ResetURL}}`, `{{.Email}}`, etc.)
   - `password_reset.txt` (plain text body, same variables)

   Use Go template syntax: `{{.FieldName}}`, `{{if .X}}...{{else}}...{{end}}`. Field names must match exported fields on the DTO.

3. **Service**: In `internal/services/email.go`, implement the new method: validate/normalize data, call `s.renderer.Render("password_reset", data)`, then `s.mailer.Send(data.Email, subject, htmlBody, textBody)`. Wrap errors with `fmt.Errorf(..., %w, err)`.

4. **Call site**: From the place that triggers the email (e.g. auth service), call `emailService.SendPasswordResetEmail(ctx, &domain.PasswordResetEmailData{...})`. No change in `cmd/api/main.go` unless you introduce a new adapter or config.

## Editing existing templates

- Template files live in `internal/adapters/email/templates/`. Naming: `<name>_subject.txt`, `<name>.html`, `<name>.txt`.
- After changing template files, rebuild; they are embedded at build time (`//go:embed templates/*` in `template_renderer.go`).

## Config and environment

- Email/SES config is in `config/config.go` (`EmailConfig`, `SESConfig`) and loaded from env: `EMAIL_PROVIDER`, `EMAIL_FROM_ADDRESS`, `EMAIL_FROM_NAME`, `AWS_SES_REGION`, `AWS_SES_ACCESS_KEY_ID`, `AWS_SES_SECRET_ACCESS_KEY`, `AWS_SES_INSECURE_SKIP_VERIFY`.
- Wiring is in `cmd/api/main.go`: build `email.MailerConfig` from `cfg.Email`, create mailer and `email.NewTemplateRenderer()`, then `services.NewEmailService(mailer, templateRenderer)`.

## Reference files

- Domain: `internal/domain/email.go`
- Adapter: `internal/adapters/email/mailer.go`, `internal/adapters/email/template_renderer.go`
- Templates: `internal/adapters/email/templates/welcome_*`
- Service: `internal/services/email.go`
- Rule: `.cursor/rules/email.mdc`
