package mailer

import (
	"context"

	"go.uber.org/zap"
)

// NoopMailer — заглушка, фактически не отправляет письма.
//
// Используется:
//   - в unit-тестах AuthService (чтобы тесты не пытались куда-то ходить),
//   - как fallback, когда SMTP_HOST не задан в конфигурации (например,
//     при локальной разработке без реального почтового аккаунта). В этом
//     случае печатает тему/получателя в лог — чтобы разработчик видел,
//     что писмо «ушло» бы.
type NoopMailer struct {
	Log *zap.Logger
}

func (n *NoopMailer) Send(_ context.Context, to, subject, htmlBody string) error {
	if n.Log != nil {
		n.Log.Info("mailer noop (skipped, no SMTP configured)",
			zap.String("to", to),
			zap.String("subject", subject),
			zap.Int("body_size", len(htmlBody)),
		)
	}
	return nil
}
