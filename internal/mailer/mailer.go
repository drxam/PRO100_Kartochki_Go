// Package mailer абстрагирует отправку email — для писем восстановления
// пароля, подтверждения регистрации и других транзакционных уведомлений.
//
// Реальная реализация (SMTPMailer) ходит на внешний SMTP-сервер;
// NoopMailer ничего не отправляет, только логирует — используется в тестах
// и как fallback, если SMTP не сконфигурирован.
package mailer

import "context"

// Mailer — единый контракт. Одного метода достаточно: тема + HTML-тело,
// получатель — конкретный адрес. Group-рассылки пока не нужны.
type Mailer interface {
	Send(ctx context.Context, to, subject, htmlBody string) error
}
