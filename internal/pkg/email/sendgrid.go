package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"
)

// SendGridConfig holds SendGrid configuration
type SendGridConfig struct {
	APIKey    string
	FromEmail string
	FromName  string
}

// SendGridClient sends emails via SendGrid API
type SendGridClient struct {
	config     SendGridConfig
	httpClient *http.Client
	templates  map[string]*template.Template
}

// NewSendGridClient creates a new SendGrid email client
func NewSendGridClient(config SendGridConfig) *SendGridClient {
	return &SendGridClient{
		config: config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		templates: make(map[string]*template.Template),
	}
}

// EmailMessage represents an email to send
type EmailMessage struct {
	To          string
	ToName      string
	Subject     string
	HTMLContent string
	TextContent string
}

// SendGridRequest represents the SendGrid API request
type SendGridRequest struct {
	Personalizations []SendGridPersonalization `json:"personalizations"`
	From             SendGridEmail             `json:"from"`
	Subject          string                    `json:"subject"`
	Content          []SendGridContent         `json:"content"`
}

type SendGridPersonalization struct {
	To []SendGridEmail `json:"to"`
}

type SendGridEmail struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

type SendGridContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// Send sends an email via SendGrid
func (c *SendGridClient) Send(ctx context.Context, msg *EmailMessage) error {
	request := SendGridRequest{
		Personalizations: []SendGridPersonalization{
			{
				To: []SendGridEmail{
					{Email: msg.To, Name: msg.ToName},
				},
			},
		},
		From: SendGridEmail{
			Email: c.config.FromEmail,
			Name:  c.config.FromName,
		},
		Subject: msg.Subject,
		Content: []SendGridContent{},
	}

	// Add HTML content first (preferred)
	if msg.HTMLContent != "" {
		request.Content = append(request.Content, SendGridContent{
			Type:  "text/html",
			Value: msg.HTMLContent,
		})
	}

	// Add plain text as fallback
	if msg.TextContent != "" {
		request.Content = append(request.Content, SendGridContent{
			Type:  "text/plain",
			Value: msg.TextContent,
		})
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.sendgrid.com/v3/mail/send", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sendgrid returned status %d", resp.StatusCode)
	}

	return nil
}

// LoadTemplate loads an HTML template
func (c *SendGridClient) LoadTemplate(name string, content string) error {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return err
	}
	c.templates[name] = tmpl
	return nil
}

// RenderTemplate renders a template with data
func (c *SendGridClient) RenderTemplate(name string, data interface{}) (string, error) {
	tmpl, ok := c.templates[name]
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// SendTemplate sends an email using a template
func (c *SendGridClient) SendTemplate(ctx context.Context, to, toName, templateName, subject string, data interface{}) error {
	html, err := c.RenderTemplate(templateName, data)
	if err != nil {
		return err
	}

	return c.Send(ctx, &EmailMessage{
		To:          to,
		ToName:      toName,
		Subject:     subject,
		HTMLContent: html,
	})
}
