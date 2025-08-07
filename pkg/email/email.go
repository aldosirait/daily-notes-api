package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"

	"daily-notes-api/internal/config"
)

type EmailService struct {
	host     string
	port     int
	username string
	password string
	from     string
}

type EmailData struct {
	To      string
	Subject string
	Body    string
	IsHTML  bool
}

type WelcomeEmailData struct {
	UserName string
	Email    string
	AppName  string
	AppURL   string
}

// NewEmailService creates a new email service
func NewEmailService(cfg *config.Config) *EmailService {
	return &EmailService{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     cfg.SMTPFrom,
	}
}

// SendEmail sends an email using SMTP
func (es *EmailService) SendEmail(emailData EmailData) error {
	// Create SMTP auth
	auth := smtp.PlainAuth("", es.username, es.password, es.host)

	// Compose message
	message := es.composeMessage(emailData)

	// Send email
	addr := fmt.Sprintf("%s:%d", es.host, es.port)
	err := smtp.SendMail(addr, auth, es.from, []string{emailData.To}, message)
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

// SendWelcomeEmail sends a welcome email to new users
func (es *EmailService) SendWelcomeEmail(userData WelcomeEmailData) error {
	// Generate HTML template
	htmlBody, err := es.generateWelcomeHTML(userData)
	if err != nil {
		return fmt.Errorf("failed to generate welcome email HTML: %w", err)
	}

	// Generate plain text version
	textBody := es.generateWelcomeText(userData)

	// Compose multipart email
	emailData := EmailData{
		To:      userData.Email,
		Subject: fmt.Sprintf("Welcome to %s! 🎉", userData.AppName),
		Body:    htmlBody,
		IsHTML:  true,
	}

	// If HTML fails, fall back to plain text
	if err := es.SendEmail(emailData); err != nil {
		emailData.Body = textBody
		emailData.IsHTML = false
		return es.SendEmail(emailData)
	}

	return nil
}

// composeMessage creates the email message with headers
func (es *EmailService) composeMessage(emailData EmailData) []byte {
	var message bytes.Buffer

	// Headers
	message.WriteString(fmt.Sprintf("From: %s\r\n", es.from))
	message.WriteString(fmt.Sprintf("To: %s\r\n", emailData.To))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", emailData.Subject))

	if emailData.IsHTML {
		message.WriteString("MIME-Version: 1.0\r\n")
		message.WriteString("Content-Type: multipart/alternative; boundary=\"boundary123\"\r\n")
		message.WriteString("\r\n")

		// Plain text part (for compatibility)
		message.WriteString("--boundary123\r\n")
		message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		message.WriteString("\r\n")
		message.WriteString(es.htmlToText(emailData.Body))
		message.WriteString("\r\n")

		// HTML part
		message.WriteString("--boundary123\r\n")
		message.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
		message.WriteString("\r\n")
		message.WriteString(emailData.Body)
		message.WriteString("\r\n")
		message.WriteString("--boundary123--\r\n")
	} else {
		message.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
		message.WriteString("\r\n")
		message.WriteString(emailData.Body)
	}

	return message.Bytes()
}

// generateWelcomeHTML generates HTML content for welcome email
func (es *EmailService) generateWelcomeHTML(data WelcomeEmailData) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Welcome to {{.AppName}}</title>
    <style>
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f8f9fa;
        }
        .container {
            background-color: white;
            border-radius: 10px;
            padding: 40px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .header h1 {
            color: #4f46e5;
            margin: 0;
            font-size: 28px;
        }
        .emoji {
            font-size: 48px;
            margin: 20px 0;
        }
        .content {
            margin: 20px 0;
        }
        .highlight {
            background-color: #f0f9ff;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #4f46e5;
            margin: 20px 0;
        }
        .button {
            display: inline-block;
            background-color: #4f46e5;
            color: white;
            padding: 12px 24px;
            text-decoration: none;
            border-radius: 6px;
            margin: 20px 0;
            font-weight: 600;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #e5e7eb;
            text-align: center;
            color: #6b7280;
            font-size: 14px;
        }
        .features {
            margin: 30px 0;
        }
        .feature {
            margin: 15px 0;
            padding: 10px 0;
        }
        .feature-icon {
            display: inline-block;
            width: 20px;
            margin-right: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="emoji">🎉</div>
            <h1>Welcome to {{.AppName}}!</h1>
            <p>Hi {{.UserName}}, we're excited to have you on board!</p>
        </div>

        <div class="content">
            <div class="highlight">
                <h3>🌟 Your account has been successfully created!</h3>
                <p>You can now start organizing your thoughts and ideas with our daily notes app.</p>
            </div>

            <div class="features">
                <h3>Here's what you can do with {{.AppName}}:</h3>
                
                <div class="feature">
                    <span class="feature-icon">📝</span>
                    <strong>Create Notes:</strong> Write and organize your daily thoughts and ideas
                </div>
                
                <div class="feature">
                    <span class="feature-icon">🏷️</span>
                    <strong>Categorize:</strong> Organize your notes with custom categories
                </div>
                
                <div class="feature">
                    <span class="feature-icon">🔍</span>
                    <strong>Search & Filter:</strong> Easily find your notes when you need them
                </div>
                
                <div class="feature">
                    <span class="feature-icon">🔒</span>
                    <strong>Secure & Private:</strong> Your notes are safe and only accessible by you
                </div>
            </div>

            <div style="text-align: center;">
                <a href="{{.AppURL}}" class="button">Get Started Now</a>
            </div>

            <p>If you have any questions or need help getting started, don't hesitate to reach out to our support team.</p>
        </div>

        <div class="footer">
            <p>Thank you for joining {{.AppName}}!</p>
            <p>This email was sent to {{.Email}}</p>
            <p>© {{.AppName}} - Your Digital Note-Taking Companion</p>
        </div>
    </div>
</body>
</html>
`

	t, err := template.New("welcome").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateWelcomeText generates plain text content for welcome email
func (es *EmailService) generateWelcomeText(data WelcomeEmailData) string {
	return fmt.Sprintf(`
Welcome to %s! 🎉

Hi %s,

We're excited to have you on board! Your account has been successfully created and you can now start organizing your thoughts and ideas with our daily notes app.

Here's what you can do with %s:

📝 Create Notes: Write and organize your daily thoughts and ideas
🏷️ Categorize: Organize your notes with custom categories  
🔍 Search & Filter: Easily find your notes when you need them
🔒 Secure & Private: Your notes are safe and only accessible by you

Get started now: %s

If you have any questions or need help getting started, don't hesitate to reach out to our support team.

Thank you for joining %s!

This email was sent to %s
© %s - Your Digital Note-Taking Companion
`, data.AppName, data.UserName, data.AppName, data.AppURL, data.AppName, data.Email, data.AppName)
}

// htmlToText converts basic HTML to plain text
func (es *EmailService) htmlToText(html string) string {
	// Simple HTML to text conversion
	text := strings.ReplaceAll(html, "<br>", "\n")
	text = strings.ReplaceAll(text, "<br/>", "\n")
	text = strings.ReplaceAll(text, "<br />", "\n")
	text = strings.ReplaceAll(text, "</p>", "\n\n")
	text = strings.ReplaceAll(text, "</div>", "\n")
	text = strings.ReplaceAll(text, "</h1>", "\n")
	text = strings.ReplaceAll(text, "</h2>", "\n")
	text = strings.ReplaceAll(text, "</h3>", "\n")

	// Remove HTML tags (basic regex replacement)
	// For production, consider using a proper HTML-to-text library
	for strings.Contains(text, "<") && strings.Contains(text, ">") {
		start := strings.Index(text, "<")
		end := strings.Index(text[start:], ">")
		if end != -1 {
			text = text[:start] + text[start+end+1:]
		} else {
			break
		}
	}

	return strings.TrimSpace(text)
}

// SendTestEmail sends a test email (useful for testing SMTP configuration)
func (es *EmailService) SendTestEmail(toEmail string) error {
	emailData := EmailData{
		To:      toEmail,
		Subject: "Test Email - SMTP Configuration",
		Body:    "This is a test email to verify SMTP configuration is working correctly.",
		IsHTML:  false,
	}

	return es.SendEmail(emailData)
}
