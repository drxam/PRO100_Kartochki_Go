package domain

import "time"

// PasswordResetToken — запись из таблицы password_reset_tokens.
// Используется при восстановлении доступа (ТЗ §4.2 «Восстановление доступа»).
type PasswordResetToken struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	Token     string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}
