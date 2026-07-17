package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"mime"
	"net/smtp"
	"os"
	"path/filepath"
)

type MailService struct {
	smtpHost string
	smtpPort string
	from     string
	appPass  string
}

func NewMailService() *MailService {
	return &MailService{
		smtpHost: "smtp.gmail.com",
		smtpPort: "587",
		from:     os.Getenv("GMAIL_USER"),     // your gmail address
		appPass:  os.Getenv("GMAIL_APP_PASS"), // 16-char app password
	}
}

type Attachment struct {
	Filename string
	Data     []byte
}

// SendMail sends an email, with optional attachments (pass zero or more).
func (m *MailService) SendMail(to, subject, body string, attachments ...Attachment) error {
	if m.from == "" || m.appPass == "" {
		return fmt.Errorf("GMAIL_USER or GMAIL_APP_PASS not set")
	}

	var buf bytes.Buffer

	boundary := "MIME-BOUNDARY-go-mail-12345"

	buf.WriteString("From: " + m.from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	if len(attachments) == 0 {
		// No attachments: keep it simple, same as before
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(body + "\r\n")
	} else {
		// Multipart message: body + attachments
		buf.WriteString("Content-Type: multipart/mixed; boundary=" + boundary + "\r\n\r\n")

		// Body part
		buf.WriteString("--" + boundary + "\r\n")
		buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
		buf.WriteString(body + "\r\n\r\n")

		// Attachment parts
		for _, att := range attachments {
			mimeType := mime.TypeByExtension(filepath.Ext(att.Filename))
			if mimeType == "" {
				mimeType = "application/octet-stream"
			}

			buf.WriteString("--" + boundary + "\r\n")
			buf.WriteString("Content-Type: " + mimeType + "\r\n")
			buf.WriteString("Content-Transfer-Encoding: base64\r\n")
			buf.WriteString("Content-Disposition: attachment; filename=\"" + att.Filename + "\"\r\n\r\n")

			encoded := base64.StdEncoding.EncodeToString(att.Data)
			// split into 76-char lines (RFC 2045 requirement)
			for i := 0; i < len(encoded); i += 76 {
				end := i + 76
				if end > len(encoded) {
					end = len(encoded)
				}
				buf.WriteString(encoded[i:end] + "\r\n")
			}
			buf.WriteString("\r\n")
		}

		buf.WriteString("--" + boundary + "--\r\n")
	}

	auth := smtp.PlainAuth("", m.from, m.appPass, m.smtpHost)
	addr := m.smtpHost + ":" + m.smtpPort

	return smtp.SendMail(addr, auth, m.from, []string{to}, buf.Bytes())
}
