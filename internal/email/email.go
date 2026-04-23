package email

// Сервис для отправки кодов подтверждения

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"traindesk/internal/config"
)

type SMTPConfig struct {
	API    string
	Sender string
}

type Sender struct {
	cfg SMTPConfig
}

func NewSender() *Sender {
	SMTPcfg := config.LoadSMTP()
	cfg := SMTPConfig{
		API:    SMTPcfg.API,
		Sender: SMTPcfg.Sender,
	}
	return &Sender{cfg: cfg}
}

// SendEmail Универсальный метод для отправки любого письма.
func (s *Sender) SendEmail(from string, to string, subject string, body string) error {
	url := "https://api.smtp2go.com/v3/email/send"

	payload := strings.NewReader("{\"to\":[\"Спуфер Матерей <alanraiskii@edu.hse.ru>\"],\"sender\":\"Dolbayeb <piska@markhse.ru>\",\"subject\":\"Piska\",\"html_body\":\"<h1>Спуфни Владимира Ваганова по братски</h1>\",\"fastaccept\":false}")

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-Smtp2go-Api-Key", "api-FBBFD637A92C423E980D0148EB9C814E")

	res, _ := http.DefaultClient.Do(req)

	// TODO
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	fmt.Println(string(body))

	return nil
}
func (s *Sender) SendVerificationEmail(toEmail, code string) error {

}
