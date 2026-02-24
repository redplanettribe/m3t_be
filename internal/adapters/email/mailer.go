package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ses"
	"github.com/aws/aws-sdk-go-v2/service/ses/types"

	"multitrackticketing/internal/domain"
)

// SESConfig holds configuration for AWS SES.
type SESConfig struct {
	Region             string
	AccessKeyID        string
	SecretAccessKey    string
	InsecureSkipVerify bool
}

// MailerConfig holds configuration for creating a mailer.
type MailerConfig struct {
	Provider    string
	FromAddress string
	FromName    string
	SES         SESConfig
}

// NewMailer creates a mailer from config. Provider "ses" uses AWS SES; "noop" or unknown uses a no-op mailer.
func NewMailer(config MailerConfig) (domain.Mailer, error) {
	switch config.Provider {
	case "ses":
		sesConfig := config.SES
		if config.SES.InsecureSkipVerify {
			log.Printf("[MAILER] WARNING: TLS certificate verification is disabled for SES. Use only in development.")
		}
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: sesConfig.InsecureSkipVerify,
					MinVersion:         tls.VersionTLS12,
				},
			},
		}
		awsCfg := aws.Config{
			Region: sesConfig.Region,
			Credentials: aws.NewCredentialsCache(
				credentials.NewStaticCredentialsProvider(
					sesConfig.AccessKeyID,
					sesConfig.SecretAccessKey,
					"",
				),
			),
			HTTPClient: httpClient,
		}
		client := ses.NewFromConfig(awsCfg)
		return &sesMailer{
			client:      client,
			fromAddress: config.FromAddress,
			fromName:    config.FromName,
		}, nil
	case "noop":
		return &noopMailer{}, nil
	default:
		log.Printf("[MAILER] Unknown email provider %q, using noop", config.Provider)
		return &noopMailer{}, nil
	}
}

type sesMailer struct {
	client      *ses.Client
	fromAddress string
	fromName    string
}

func (s *sesMailer) Send(to, subject, html, text string) error {
	source := s.fromAddress
	if s.fromName != "" {
		source = fmt.Sprintf("%s <%s>", s.fromName, s.fromAddress)
	}
	input := &ses.SendEmailInput{
		Source: aws.String(source),
		Destination: &types.Destination{
			ToAddresses: []string{to},
		},
		Message: &types.Message{
			Subject: &types.Content{
				Data:    aws.String(subject),
				Charset: aws.String("UTF-8"),
			},
			Body: &types.Body{},
		},
	}
	if html != "" {
		input.Message.Body.Html = &types.Content{
			Data:    aws.String(html),
			Charset: aws.String("UTF-8"),
		}
	}
	if text != "" {
		input.Message.Body.Text = &types.Content{
			Data:    aws.String(text),
			Charset: aws.String("UTF-8"),
		}
	}
	ctx := context.Background()
	result, err := s.client.SendEmail(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send email via SES: %w", err)
	}
	log.Printf("[MAILER] Email sent via SES. MessageID: %s", aws.ToString(result.MessageId))
	return nil
}

type noopMailer struct{}

func (n *noopMailer) Send(to, subject, html, text string) error {
	log.Println("[MAILER] Email would be sent (noop)", "to", to, "subject", subject)
	return nil
}
