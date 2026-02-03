package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
)

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO users (email, password_hash, username, avatar_url, role)
		VALUES ($1, $2, $3, $4, COALESCE($5, 'user'))
		RETURNING id, created_at, updated_at`
	return r.db.Pool.QueryRow(ctx, query,
		u.Email, u.PasswordHash, u.Username, u.AvatarURL, u.Role,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	query := `SELECT id, email, password_hash, username, avatar_url, role, created_at, updated_at
		FROM users WHERE id = $1`
	var u domain.User
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Username, &u.AvatarURL, &u.Role,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT id, email, password_hash, username, avatar_url, role, created_at, updated_at
		FROM users WHERE email = $1`
	var u domain.User
	err := r.db.Pool.QueryRow(ctx, query, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Username, &u.AvatarURL, &u.Role,
		&u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	query := `UPDATE users SET email=$2, username=$3, avatar_url=$4, role=$5, updated_at=NOW()
		WHERE id=$1 RETURNING updated_at`
	return r.db.Pool.QueryRow(ctx, query, u.ID, u.Email, u.Username, u.AvatarURL, u.Role).Scan(&u.UpdatedAt)
}
