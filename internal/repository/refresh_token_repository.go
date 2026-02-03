package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
)

type RefreshTokenRepository struct {
	db *DB
}

func NewRefreshTokenRepository(db *DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, rt *domain.RefreshToken) error {
	query := `INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES ($1, $2, $3) RETURNING id, created_at`
	return r.db.Pool.QueryRow(ctx, query, rt.UserID, rt.Token, rt.ExpiresAt).Scan(&rt.ID, &rt.CreatedAt)
}

func (r *RefreshTokenRepository) GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	query := `SELECT id, user_id, token, expires_at, created_at FROM refresh_tokens WHERE token = $1`
	var rt domain.RefreshToken
	err := r.db.Pool.QueryRow(ctx, query, token).Scan(&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt, &rt.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &rt, nil
}

func (r *RefreshTokenRepository) DeleteByToken(ctx context.Context, token string) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token = $1`, token)
	return err
}

func (r *RefreshTokenRepository) DeleteByUserID(ctx context.Context, userID int) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE user_id = $1`, userID)
	return err
}
