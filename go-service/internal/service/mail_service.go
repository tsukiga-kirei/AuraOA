package service

import (
	"fmt"
	"strings"

	"oa-smart-audit/go-service/internal/pkg/mail"
	"oa-smart-audit/go-service/internal/repository"
)

// MailService 系统级邮件服务
type MailService struct {
	configRepo *repository.SystemConfigRepo
}

func NewMailService(repo *repository.SystemConfigRepo) *MailService {
	return &MailService{configRepo: repo}
}

// SendReport 发送周报/日报至指定邮件地址
func (s *MailService) SendReport(to string, subject, content string) error {
	host, _ := s.configRepo.FindByKey("system.smtp_host")
	portStr, _ := s.configRepo.FindByKey("system.smtp_port")
	username, _ := s.configRepo.FindByKey("system.smtp_username")
	password, _ := s.configRepo.FindByKey("system.smtp_password")
	sender, _ := s.configRepo.FindByKey("system.smtp_sender")
	sslStr, _ := s.configRepo.FindByKey("system.smtp_ssl")

	if host == "" || username == "" {
		return fmt.Errorf("SMTP 配置不完整，请在系统设置中完善邮件地址及认证信息")
	}

	cfg := mail.Config{
		Host:     host,
		Port:     mail.ParsePort(portStr),
		Username: username,
		Password: password,
		From:     sender,
		UseSSL:    strings.ToLower(sslStr) == "true",
	}

	if cfg.From == "" {
		cfg.From = username // 若未设置发件人，默认为用户名
	}

	mailer := mail.NewMailer(cfg)
	return mailer.Send(to, subject, content)
}
