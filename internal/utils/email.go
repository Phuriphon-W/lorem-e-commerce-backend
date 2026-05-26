package utils

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/smtp"
	"time"
)

//go:embed templates/reset_password.html templates/styles.css
var templateFS embed.FS

type EmailService interface {
	SendResetPasswordEmail(toEmail, userName, resetLink string) error
}

type smtpEmailService struct {
	host     string
	port     int
	user     string
	password string
	from     string
}

func NewSMTPEmailService(host string, port int, user, password, from string) EmailService {
	return &smtpEmailService{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     from,
	}
}

func (s *smtpEmailService) SendResetPasswordEmail(toEmail, userName, resetLink string) error {
	subject := "[Lorem] Password Reset Request"

	// Read the CSS file into a string
	cssBytes, err := templateFS.ReadFile("templates/styles.css")
	if err != nil {
		return fmt.Errorf("failed to read css: %w", err)
	}

	// Add template.CSS to tell Go it's safe to inject as raw CSS (prevents auto-escaping)
	templateData := struct {
		UserName  string
		ResetLink string
		Year      int
		CSS       template.CSS
	}{
		UserName:  userName,
		ResetLink: resetLink,
		Year:      time.Now().Year(),
		CSS:       template.CSS(cssBytes),
	}

	// Parse the embedded HTML file
	tmpl, err := template.ParseFS(templateFS, "templates/reset_password.html")
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	// Execute the template
	var htmlBody bytes.Buffer
	if err := tmpl.Execute(&htmlBody, templateData); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// Construct the RFC 822 format email
	var body bytes.Buffer
	body.WriteString(fmt.Sprintf("From: %s\r\n", s.from))
	body.WriteString(fmt.Sprintf("To: %s\r\n", toEmail))
	body.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	body.WriteString("MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n")
	body.Write(htmlBody.Bytes())

	// Send the email
	var auth smtp.Auth
	if s.user != "" && s.password != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	err = smtp.SendMail(addr, auth, s.from, []string{toEmail}, body.Bytes())
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}
