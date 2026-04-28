package mailer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	mail "gopkg.in/mail.v2"
)

// SMTPConfig — параметры подключения к SMTP-серверу.
//
// Примеры:
//
//	Gmail:  Host=smtp.gmail.com, Port=587, TLSMode=starttls
//	Yandex: Host=smtp.yandex.ru, Port=465, TLSMode=ssl
//	Mail.ru: Host=smtp.mail.ru,  Port=465, TLSMode=ssl
//
// Username — это login почты (например, you@gmail.com). Password — App
// Password из настроек безопасности аккаунта (НЕ обычный пароль), 2FA на
// аккаунте обязательна.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string        // обычно совпадает с Username; иначе сервер может счесть письмо спуфингом
	TLSMode  string        // "starttls" | "ssl" | "none"
	Timeout  time.Duration // 0 → 10s
}

// SMTPMailer — реализация Mailer поверх gopkg.in/mail.v2.
type SMTPMailer struct {
	dialer  *mail.Dialer
	from    string
	timeout time.Duration
	log     *zap.Logger
}

// NewSMTP строит mailer из конфигурации. Если cfg некорректна — возвращает ошибку.
func NewSMTP(cfg SMTPConfig, log *zap.Logger) (*SMTPMailer, error) {
	if cfg.Host == "" || cfg.Port == 0 {
		return nil, errors.New("mailer/smtp: host and port are required")
	}
	if cfg.From == "" {
		// Большинство SMTP-серверов отбрасывают письмо без From; для удобства
		// подставляем username, если From не задан явно.
		cfg.From = cfg.Username
	}
	if cfg.From == "" {
		return nil, errors.New("mailer/smtp: from address is required")
	}

	d := mail.NewDialer(cfg.Host, cfg.Port, cfg.Username, cfg.Password)
	switch strings.ToLower(cfg.TLSMode) {
	case "ssl":
		d.SSL = true
	case "none":
		d.SSL = false
		// StartTLSPolicy по умолчанию = OpportunisticStartTLS — подходит.
	default: // "starttls" и пустое
		d.SSL = false
		d.StartTLSPolicy = mail.MandatoryStartTLS
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &SMTPMailer{dialer: d, from: cfg.From, timeout: timeout, log: log}, nil
}

// Send блокируется не дольше timeout (по умолчанию 10 сек). Любые ошибки
// SMTP оборачиваются и логируются — caller обычно может их игнорировать
// (например, в forgot-password нельзя выдавать факт сбоя клиенту).
func (s *SMTPMailer) Send(ctx context.Context, to, subject, htmlBody string) error {
	m := mail.NewMessage()
	m.SetHeader("From", s.from)
	m.SetHeader("To", to)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html; charset=UTF-8", htmlBody)

	done := make(chan error, 1)
	go func() { done <- s.dialer.DialAndSend(m) }()

	select {
	case err := <-done:
		if err != nil {
			s.log.Warn("smtp send failed",
				zap.String("to", to),
				zap.String("subject", subject),
				zap.Error(err))
			return fmt.Errorf("smtp send: %w", err)
		}
		s.log.Info("smtp sent",
			zap.String("to", to),
			zap.String("subject", subject))
		return nil
	case <-time.After(s.timeout):
		s.log.Warn("smtp send timeout",
			zap.String("to", to),
			zap.Duration("timeout", s.timeout))
		return fmt.Errorf("smtp send: timeout after %s", s.timeout)
	case <-ctx.Done():
		return ctx.Err()
	}
}
