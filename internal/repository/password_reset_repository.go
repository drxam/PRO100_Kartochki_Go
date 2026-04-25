package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
)

type PasswordResetRepository struct {
	db *DB
}

func NewPasswordResetRepository(db *DB) *PasswordResetRepository {
	return &PasswordResetRepository{db: db}
}

func (r *PasswordResetRepository) Create(ctx context.Context, prt *domain.PasswordResetToken) error {
	query := `INSERT INTO password_reset_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3) RETURNING id, created_at`
	return r.db.Pool.QueryRow(ctx, query, prt.UserID, prt.Token, prt.ExpiresAt).
		Scan(&prt.ID, &prt.CreatedAt)
}

func (r *PasswordResetRepository) GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	query := `SELECT id, user_id, token, expires_at, used_at, created_at
		FROM password_reset_tokens WHERE token = $1`
	var prt domain.PasswordResetToken
	err := r.db.Pool.QueryRow(ctx, query, token).Scan(
		&prt.ID, &prt.UserID, &prt.Token, &prt.ExpiresAt, &prt.UsedAt, &prt.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &prt, nil
}

// MarkUsed помечает токен использованным (одноразовость).
func (r *PasswordResetRepository) MarkUsed(ctx context.Context, id int) error {
	_, err := r.db.Pool.Exec(ctx, `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`, id)
	return err
}

// InvalidateActiveForUser помечает все активные (не использованные) токены
// конкретного пользователя как использованные. Вызывается при выдаче нового
// токена сброса, чтобы старые ссылки сразу переставали работать.
func (r *PasswordResetRepository) InvalidateActiveForUser(ctx context.Context, userID int) error {
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE password_reset_tokens SET used_at = NOW()
		 WHERE user_id = $1 AND used_at IS NULL`, userID)
	return err
}

// DeleteExpired удаляет полностью отработавшие токены сброса:
// либо истёкшие, либо использованные более `retentionForUsed` назад.
// Использованные держим короткое время для возможного аудита.
func (r *PasswordResetRepository) DeleteExpired(ctx context.Context, retentionForUsed time.Duration) (int64, error) {
	threshold := time.Now().Add(-retentionForUsed)
	tag, err := r.db.Pool.Exec(ctx,
		`DELETE FROM password_reset_tokens
		 WHERE expires_at < NOW() OR (used_at IS NOT NULL AND used_at < $1)`,
		threshold)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
