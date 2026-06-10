package mail

import (
	"fmt"
	"net/smtp"
	"photo-viewer-server/internal/config"
)

type MailService struct {
	smtpUser string
	smtpHost string
	smtpPassword string
	smtpFromAddress string
}

func NewMailService(config *config.Config) MailService {
	return MailService{
		smtpUser: config.SmtpUser,
		smtpHost: config.SmtpHost,
		smtpPassword: config.SmtpPassword,
		smtpFromAddress: config.SmtpFromAddress,
	}
}

func (ms *MailService) SendMail(to, subject, body string) error {
	auth := smtp.PlainAuth("", ms.smtpUser, ms.smtpPassword, ms.smtpHost)
	msg := make([]byte, 0, 512)
	msg = fmt.Appendf(msg, "From: %s\r\n", ms.smtpFromAddress)
	msg = fmt.Appendf(msg, "To: %s\r\n", to)
	msg = fmt.Appendf(msg, "Subject: %s\r\n", subject)
	msg = fmt.Appendf(msg, "\r\n%s\r\n", body)
	return smtp.SendMail(fmt.Sprintf("%s:587", ms.smtpHost), auth, ms.smtpFromAddress, []string{to}, msg)
}
