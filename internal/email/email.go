package email

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"time"
	"traindesk/internal/config"
)

type SMTPConfig struct {
	API    string
	Sender string
}

type Sender struct {
	APIKey string
	Sender string
	Client *http.Client
}

func NewSender() *Sender {
	cfg := config.LoadSMTP()

	return &Sender{
		APIKey: cfg.API,
		Sender: cfg.Sender,
		Client: &http.Client{Timeout: 15 * time.Second},
	}
}

// SendEmail общий метод, который инкапсулирует отправку текста по API
func (s *Sender) SendEmail(from, name, to, subject, body string) error {
	url := "https://api.smtp2go.com/v3/email/send"

	// Используем map для гибкости, так как нам не нужны вложения (attachments) во всех письмах
	payload := map[string]interface{}{
		"to":        name + " " + "<" + to + ">",
		"sender":    from,
		"subject":   subject,
		"html_body": body,
		"text_body": "Пожалуйста, используйте почтовый клиент с поддержкой HTML",
	}

	jsonValue, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("ошибка подготовки JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonValue))
	if err != nil {
		return err
	}

	// Устанавливаем заголовки согласно документации
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("accept", "application/json")
	req.Header.Add("X-Smtp2go-Api-Key", s.APIKey)

	res, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка отправки запроса: %w", err)
	}
	defer res.Body.Close()

	// Читаем тело ответа для логирования или обработки специфических ошибок
	respBody, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("API ошибка (статус %d): %s", res.StatusCode, string(respBody))
	}

	return nil
}

func generateVerificationCodeHTML(code string) (string, error) {
	const tmplString = `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Подтверждение TrainDesk</title>
</head>
<body style="margin: 0; padding: 0; font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; background-color: #f9fafb; color: #1f2937;">
    <div style="max-width: 600px; margin: 40px auto; background-color: #ffffff; border-radius: 12px; overflow: hidden; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);">
        
        <!-- Header -->
        <div style="background-color: #2563eb; padding: 30px; text-align: center;">
            <h1 style="margin: 0; color: #ffffff; font-size: 28px; font-weight: 800; letter-spacing: -0.5px;">TrainDesk</h1>
        </div>

        <!-- Content -->
        <div style="padding: 40px 30px; line-height: 1.6;">
            <h2 style="margin-top: 0; color: #111827; font-size: 20px;">Подтверждение действия</h2>
            <p style="font-size: 16px; color: #4b5563;">Здравствуйте! Используйте этот код для подтверждения вашего действия в системе <strong>TrainDesk</strong>:</p>
            
            <div style="margin: 30px 0; text-align: center;">
                <div style="display: inline-block; padding: 15px 40px; background-color: #f3f4f6; border: 2px solid #e5e7eb; border-radius: 8px; font-size: 32px; font-weight: bold; color: #2563eb; letter-spacing: 4px;">
                    {{.Code}}
                </div>
            </div>

            <p style="font-size: 14px; color: #6b7280; border-top: 1px solid #f3f4f6; padding-top: 20px;">
                Если вы не совершали никаких действий в <strong>TrainDesk</strong>, просто проигнорируйте это письмо. Ваш аккаунт в безопасности.
            </p>
        </div>

        <!-- Footer -->
        <div style="background-color: #f9fafb; padding: 20px; text-align: center; border-top: 1px solid #f3f4f6;">
            <p style="margin: 0; font-size: 12px; color: #9ca3af; text-transform: uppercase; letter-spacing: 1px;">
                Это автоматическое уведомление
            </p>
            <p style="margin: 5px 0 0; font-size: 13px; color: #9ca3af;">
                Пожалуйста, не отвечайте на это письмо.
            </p>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("email").Parse(tmplString)
	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer
	data := struct{ Code string }{Code: code}

	if err := tmpl.Execute(&tpl, data); err != nil {
		return "", err
	}

	return tpl.String(), nil
}

func (s *Sender) SendEmailVerificationCode(code, to, name string) error {
	subject, err := generateVerificationCodeHTML(code)
	if err != nil {
		return err
	}

	if err := s.SendEmail(s.Sender, name, to, "Код подтверждения почты", subject); err != nil {
		return err
	}

	return nil
}
