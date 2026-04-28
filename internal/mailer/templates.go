package mailer

import (
	"fmt"
	"net/url"
)

// PasswordResetSubject — тема письма для сброса пароля.
const PasswordResetSubject = "PRO100_Карточки — восстановление пароля"

// PasswordResetHTML формирует HTML-тело письма со ссылкой для сброса.
// appBaseURL — публичный URL фронтенда (или сервиса), на который уйдёт
// пользователь, кликнув по ссылке. Токен передаётся как query-параметр.
func PasswordResetHTML(appBaseURL, token string) string {
	link := fmt.Sprintf("%s/reset-password?token=%s",
		appBaseURL, url.QueryEscape(token))

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="ru">
<head><meta charset="UTF-8"></head>
<body style="font-family: -apple-system, Segoe UI, Roboto, sans-serif; max-width: 540px; margin: 0 auto; padding: 24px; color: #1f2328;">
  <h2 style="margin-top: 0;">Восстановление пароля</h2>
  <p>Здравствуйте! Вы (или кто-то от вашего имени) запросил сброс пароля
     для учётной записи в приложении «PRO100_Карточки».</p>
  <p>Чтобы задать новый пароль, перейдите по ссылке:</p>
  <p><a href="%[1]s" style="display: inline-block; padding: 12px 24px; background: #007AFF; color: white; text-decoration: none; border-radius: 8px;">Сбросить пароль</a></p>
  <p style="font-size: 13px; color: #57606a;">Или скопируйте ссылку:<br>
     <code style="word-break: break-all;">%[1]s</code></p>
  <p style="font-size: 13px; color: #57606a;">Ссылка действительна <b>1 час</b>
     и может быть использована только один раз.</p>
  <hr style="border: none; border-top: 1px solid #d0d7de; margin: 24px 0;">
  <p style="font-size: 12px; color: #57606a;">Если вы не запрашивали сброс пароля —
     просто проигнорируйте это письмо, ничего не произойдёт.</p>
</body>
</html>`, link)
}
