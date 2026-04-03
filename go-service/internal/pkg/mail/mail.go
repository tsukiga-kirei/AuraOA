package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
)

// Config 邮件服务配置
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseSSL   bool
}

// Mailer 邮件发送器
type Mailer struct {
	config Config
}

func NewMailer(cfg Config) *Mailer {
	return &Mailer{config: cfg}
}

// Send 发送邮件（支持 HTML 内容，支持逗号分隔的多个收件人）
func (m *Mailer) Send(to string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)

	// 处理多个收件人
	toParts := strings.Split(to, ",")
	var recipients []string
	for _, p := range toParts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			recipients = append(recipients, trimmed)
		}
	}
	if len(recipients) == 0 {
		return fmt.Errorf("no valid recipients")
	}

	header := make(map[string]string)
	header["From"] = m.config.From
	header["To"] = strings.Join(recipients, ", ")
	header["Subject"] = subject
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range header {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	if m.config.UseSSL {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         m.config.Host,
		}
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return err
		}
		defer conn.Close()

		client, err := smtp.NewClient(conn, m.config.Host)
		if err != nil {
			return err
		}
		defer client.Quit()

		if err = client.Auth(auth); err != nil {
			return err
		}

		if err = client.Mail(m.config.From); err != nil {
			return err
		}

		for _, rcpt := range recipients {
			if err = client.Rcpt(rcpt); err != nil {
				return err
			}
		}

		w, err := client.Data()
		if err != nil {
			return err
		}

		_, err = w.Write([]byte(message))
		if err != nil {
			return err
		}

		return w.Close()
	}

	return smtp.SendMail(addr, auth, m.config.From, recipients, []byte(message))
}

// ParsePort 将字符串端口解析为 int，默认 465
func ParsePort(p string) int {
	port, err := strconv.Atoi(p)
	if err != nil {
		return 465
	}
	return port
}
