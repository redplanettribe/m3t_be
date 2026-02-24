package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	texttemplate "text/template"

	"multitrackticketing/internal/domain"
)

//go:embed templates/*
var templateFS embed.FS

// templateRenderer implements domain.EmailTemplateRenderer using embedded template files.
type templateRenderer struct{}

// NewTemplateRenderer returns an EmailTemplateRenderer that loads templates from the embedded templates folder.
func NewTemplateRenderer() domain.EmailTemplateRenderer {
	return &templateRenderer{}
}

// Render executes the named template (e.g. "welcome") with data and returns subject, html, and text bodies.
func (r *templateRenderer) Render(templateName string, data interface{}) (subject, htmlBody, textBody string, err error) {
	subject, err = r.renderFile(templateName+"_subject.txt", data, false)
	if err != nil {
		return "", "", "", fmt.Errorf("render subject: %w", err)
	}
	htmlBody, err = r.renderFile(templateName+".html", data, true)
	if err != nil {
		return "", "", "", fmt.Errorf("render html: %w", err)
	}
	textBody, err = r.renderFile(templateName+".txt", data, false)
	if err != nil {
		return "", "", "", fmt.Errorf("render text: %w", err)
	}
	return strings.TrimSpace(subject), htmlBody, textBody, nil
}

func (r *templateRenderer) renderFile(name string, data interface{}, html bool) (string, error) {
	raw, err := templateFS.ReadFile("templates/" + name)
	if err != nil {
		return "", err
	}
	tmplStr := string(raw)
	var buf bytes.Buffer
	if html {
		t, err := template.New(name).Parse(tmplStr)
		if err != nil {
			return "", err
		}
		if err := t.Execute(&buf, data); err != nil {
			return "", err
		}
	} else {
		t, err := texttemplate.New(name).Parse(tmplStr)
		if err != nil {
			return "", err
		}
		if err := t.Execute(&buf, data); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}
