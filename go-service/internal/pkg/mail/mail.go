// Package mail 提供 SMTP 邮件发送功能，支持 SSL/TLS 和普通连接两种模式。
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

// NewMailer 创建邮件发送器实例。
func NewMailer(cfg Config) *Mailer {
	return &Mailer{config: cfg}
}

// Send 发送 HTML 邮件，支持逗号分隔的多个收件人地址。
func (m *Mailer) Send(to string, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", m.config.Host, m.config.Port)
	auth := smtp.PlainAuth("", m.config.Username, m.config.Password, m.config.Host)

	// 解析并过滤收件人列表
	toParts := strings.Split(to, ",")
	var recipients []string
	for _, p := range toParts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			recipients = append(recipients, trimmed)
		}
	}
	if len(recipients) == 0 {
		return fmt.Errorf("收件人列表为空")
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
		// SSL 模式：先建立 TLS 连接，再创建 SMTP 客户端
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

	// 普通模式（STARTTLS 由 smtp.SendMail 自动协商）
	return smtp.SendMail(addr, auth, m.config.From, recipients, []byte(message))
}

// ParsePort 将字符串端口号解析为整数，解析失败时返回默认值 465。
func ParsePort(p string) int {
	port, err := strconv.Atoi(p)
	if err != nil {
		return 465
	}
	return port
}
