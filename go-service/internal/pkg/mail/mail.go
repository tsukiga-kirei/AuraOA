package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"
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

// Send 发送邮件（支持 HTML 内容）
func (m *Mailer) Send(to string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)

	header := make(map[string]string)
	header["From"] = m.config.From
	header["To"] = to
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

		if err = client.Rcpt(to); err != nil {
			return err
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

	return smtp.SendMail(addr, auth, m.config.From, []string{to}, []byte(message))
}

// ParsePort 将字符串端口解析为 int，默认 465
func ParsePort(p string) int {
	port, err := strconv.Atoi(p)
	if err != nil {
		return 465
	}
	return port
}
