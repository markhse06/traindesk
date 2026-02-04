package email

// Сервис для отправки кодов подтверждения

import (
	"fmt"
	"net/smtp"
)

type SMTPConfig struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
}

type Sender struct {
	cfg SMTPConfig
}

func NewSender(cfg SMTPConfig) *Sender {
	return &Sender{cfg: cfg}
}

func (s *Sender) SendVerificationEmail(toEmail, code string) error {
	// Простейшее текстовое письмо.
	subject := "TrainDesk: подтверждение почты"
	body := fmt.Sprintf("Ваш код подтверждения: %s", code)

	msg := []byte(
		"To: " + toEmail + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"\r\n" +
			body + "\r\n",
	)

	addr := s.cfg.Host + ":" + s.cfg.Port

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	return smtp.SendMail(addr, auth, s.cfg.From, []string{toEmail}, msg)
}
