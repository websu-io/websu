package api

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/smtp"
	"os"
	"path/filepath"
)

var (
	SmtpHost     = ""
	SmtpPort     = 465
	SmtpUsername = ""
	SmtpPassword = ""
	FromEmail    = "info@websu.io"
)

// used for tests
var sendEmail = smtp.SendMail

func (report *Report) SendEmail() error {
	// Set up authentication information.
	auth := smtp.PlainAuth("", SmtpUsername, SmtpPassword, SmtpHost)

	// Connect to the server, authenticate, set the sender and recipient,
	// and send the email all in one step.
	from := fmt.Sprintf("From: Websu <%s>\n", FromEmail)
	to := fmt.Sprintf("To: %s\n", report.Email)
	subject := fmt.Sprintf("Subject: Websu: Performance report for %s\n", report.URL)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	cwd, _ := os.Getwd()
	templatePath := filepath.Join(cwd, "./templates/email-template.html")
	t, err := template.ParseFiles(templatePath)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, report); err != nil {
		return err
	}
	body := buf.Bytes()
	msg := append([]byte(from+to+subject+mime), body...)
	toArr := []string{report.Email}
	server := fmt.Sprintf("%s:%d", SmtpHost, SmtpPort)
	err = sendEmail(server, auth, FromEmail, toArr, msg)
	if err != nil {
		return err
	}
	log.WithField("Report.ID", report.ID).Info("Email was sent for report")
	return nil
}
