package email

import (
	"bytes"
	"context"
	"html/template"
	"sync"

	"github.com/rs/zerolog/log"
)

// Sender interface for sending emails
type Sender interface {
	Send(ctx context.Context, msg *EmailMessage) error
	SendTemplate(ctx context.Context, to, toName, templateName, subject string, data interface{}) error
}

// Service handles email sending with templates
type Service struct {
	client       *SendGridClient
	templates    map[string]*template.Template
	baseTemplate *template.Template
	queue        chan *QueuedEmail
	wg           sync.WaitGroup
}

// QueuedEmail represents an email in the send queue
type QueuedEmail struct {
	To           string
	ToName       string
	Subject      string
	TemplateName string
	Data         interface{}
}

// NewService creates email service
func NewService(config SendGridConfig) *Service {
	s := &Service{
		client:    NewSendGridClient(config),
		templates: make(map[string]*template.Template),
		queue:     make(chan *QueuedEmail, 100),
	}

	// Load base template
	s.baseTemplate, _ = template.New("base").Parse(BaseTemplate)

	// Load all templates
	s.loadTemplates()

	// Start async worker
	s.wg.Add(1)
	go s.worker()

	return s
}

// loadTemplates loads all email templates
func (s *Service) loadTemplates() {
	templates := map[string]string{
		"response_accepted": ResponseAcceptedTemplate,
		"response_rejected": ResponseRejectedTemplate,
		"new_response":      NewResponseTemplate,
		"new_message":       NewMessageTemplate,
		"casting_expiring":  CastingExpiringTemplate,
		"welcome":           WelcomeTemplate,
		"lead_approved":     LeadApprovedTemplate,
		"lead_rejected":     LeadRejectedTemplate,
		"digest":            DigestTemplate,
	}

	for name, content := range templates {
		tmpl, err := template.New(name).Parse(content)
		if err != nil {
			log.Error().Err(err).Str("template", name).Msg("Failed to parse email template")
			continue
		}
		s.templates[name] = tmpl
	}
}

// worker processes queued emails asynchronously
func (s *Service) worker() {
	defer s.wg.Done()

	for email := range s.queue {
		ctx := context.Background()
		if err := s.send(ctx, email); err != nil {
			log.Error().Err(err).
				Str("to", email.To).
				Str("template", email.TemplateName).
				Msg("Failed to send email")
		}
	}
}

// send actually sends the email
func (s *Service) send(ctx context.Context, email *QueuedEmail) error {
	// Render template
	tmpl, ok := s.templates[email.TemplateName]
	if !ok {
		log.Warn().Str("template", email.TemplateName).Msg("Template not found")
		return nil
	}

	var contentBuf bytes.Buffer
	if err := tmpl.Execute(&contentBuf, email.Data); err != nil {
		return err
	}

	// Wrap in base template
	var htmlBuf bytes.Buffer
	if err := s.baseTemplate.Execute(&htmlBuf, map[string]interface{}{
		"Content": template.HTML(contentBuf.String()),
	}); err != nil {
		return err
	}

	return s.client.Send(ctx, &EmailMessage{
		To:          email.To,
		ToName:      email.ToName,
		Subject:     email.Subject,
		HTMLContent: htmlBuf.String(),
	})
}

// Queue adds an email to the async send queue
func (s *Service) Queue(to, toName, templateName, subject string, data interface{}) {
	select {
	case s.queue <- &QueuedEmail{
		To:           to,
		ToName:       toName,
		Subject:      subject,
		TemplateName: templateName,
		Data:         data,
	}:
	default:
		log.Warn().Str("to", to).Msg("Email queue full, dropping email")
	}
}

// SendSync sends an email synchronously (blocking)
func (s *Service) SendSync(ctx context.Context, to, toName, templateName, subject string, data interface{}) error {
	return s.send(ctx, &QueuedEmail{
		To:           to,
		ToName:       toName,
		Subject:      subject,
		TemplateName: templateName,
		Data:         data,
	})
}

// Close stops the email worker
func (s *Service) Close() {
	close(s.queue)
	s.wg.Wait()
}

// --- Convenience methods for specific emails ---

// SendResponseAccepted sends acceptance notification
func (s *Service) SendResponseAccepted(to, toName, modelName, castingTitle, employerName, castingURL string) {
	s.Queue(to, toName, "response_accepted", "🎉 Вас приняли на кастинг!", map[string]string{
		"ModelName":    modelName,
		"CastingTitle": castingTitle,
		"EmployerName": employerName,
		"CastingURL":   castingURL,
	})
}

// SendResponseRejected sends rejection notification
func (s *Service) SendResponseRejected(to, toName, castingTitle, castingsURL string) {
	s.Queue(to, toName, "response_rejected", "Заявка на кастинг", map[string]string{
		"CastingTitle": castingTitle,
		"CastingsURL":  castingsURL,
	})
}

// SendNewResponse sends new response notification to employer
func (s *Service) SendNewResponse(to, toName, castingTitle, modelName, responseURL string) {
	s.Queue(to, toName, "new_response", "📩 Новый отклик на кастинг", map[string]string{
		"CastingTitle": castingTitle,
		"ModelName":    modelName,
		"ResponseURL":  responseURL,
	})
}

// SendNewMessage sends new message notification
func (s *Service) SendNewMessage(to, toName, senderName, preview, chatURL string) {
	s.Queue(to, toName, "new_message", "💬 Новое сообщение от "+senderName, map[string]string{
		"SenderName":     senderName,
		"MessagePreview": preview,
		"ChatURL":        chatURL,
	})
}

// SendWelcome sends welcome email to new user
func (s *Service) SendWelcome(to, toName, userName, role, dashboardURL string) {
	s.Queue(to, toName, "welcome", "Добро пожаловать в MWork!", map[string]string{
		"UserName":     userName,
		"Role":         role,
		"DashboardURL": dashboardURL,
	})
}

// SendLeadApproved sends approval notification to company
func (s *Service) SendLeadApproved(to, contactName, companyName, email, tempPassword, loginURL string) {
	s.Queue(to, contactName, "lead_approved", "✅ Ваша заявка одобрена!", map[string]string{
		"ContactName":  contactName,
		"CompanyName":  companyName,
		"Email":        email,
		"TempPassword": tempPassword,
		"LoginURL":     loginURL,
	})
}

// SendLeadRejected sends rejection notification to company
func (s *Service) SendLeadRejected(to, contactName, companyName, reason string) {
	s.Queue(to, contactName, "lead_rejected", "Заявка рассмотрена", map[string]string{
		"ContactName": contactName,
		"CompanyName": companyName,
		"Reason":      reason,
	})
}
