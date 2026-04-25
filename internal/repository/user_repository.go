package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
)

// ErrUserNotFound — пользователь не найден или мягко удалён.
var ErrUserNotFound = errors.New("пользователь не найден")

type UserRepository struct {
	db *DB
}

func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// userColumns — общий список колонок для SELECT.
const userColumns = `id, email, password_hash, username, avatar_url, role,
		is_blocked, blocked_at, deleted_at, token_version, created_at, updated_at`

func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.Username, &u.AvatarURL, &u.Role,
		&u.IsBlocked, &u.BlockedAt, &u.DeletedAt, &u.TokenVersion, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	query := `INSERT INTO users (email, password_hash, username, avatar_url, role)
		VALUES ($1, $2, $3, $4, COALESCE($5, 'user'))
		RETURNING id, is_blocked, blocked_at, deleted_at, token_version, created_at, updated_at`
	return r.db.Pool.QueryRow(ctx, query,
		u.Email, u.PasswordHash, u.Username, u.AvatarURL, u.Role,
	).Scan(&u.ID, &u.IsBlocked, &u.BlockedAt, &u.DeletedAt, &u.TokenVersion, &u.CreatedAt, &u.UpdatedAt)
}

// GetByID возвращает только активных пользователей (deleted_at IS NULL).
func (r *UserRepository) GetByID(ctx context.Context, id int) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1 AND deleted_at IS NULL`
	return scanUser(r.db.Pool.QueryRow(ctx, query, id))
}

// GetByIDIncludingDeleted — версия для админских операций (видит мягко удалённых).
func (r *UserRepository) GetByIDIncludingDeleted(ctx context.Context, id int) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`
	return scanUser(r.db.Pool.QueryRow(ctx, query, id))
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1 AND deleted_at IS NULL`
	return scanUser(r.db.Pool.QueryRow(ctx, query, email))
}

func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	query := `UPDATE users SET email=$2, username=$3, avatar_url=$4, role=$5, updated_at=NOW()
		WHERE id=$1 AND deleted_at IS NULL RETURNING updated_at`
	return r.db.Pool.QueryRow(ctx, query, u.ID, u.Email, u.Username, u.AvatarURL, u.Role).Scan(&u.UpdatedAt)
}

// SetBlocked включает или снимает блокировку учётной записи.
func (r *UserRepository) SetBlocked(ctx context.Context, id int, blocked bool) error {
	query := `UPDATE users SET is_blocked = $2,
		blocked_at = CASE WHEN $2 THEN NOW() ELSE NULL END,
		updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.db.Pool.Exec(ctx, query, id, blocked)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SoftDelete помечает пользователя удалённым (deleted_at = NOW()).
func (r *UserRepository) SoftDelete(ctx context.Context, id int) error {
	query := `UPDATE users SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// SetRole меняет роль пользователя.
func (r *UserRepository) SetRole(ctx context.Context, id int, role string) error {
	query := `UPDATE users SET role = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.db.Pool.Exec(ctx, query, id, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdatePassword обновляет password_hash. Используется при сбросе пароля.
func (r *UserRepository) UpdatePassword(ctx context.Context, id int, passwordHash string) error {
	query := `UPDATE users SET password_hash = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.db.Pool.Exec(ctx, query, id, passwordHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

// List возвращает страницу пользователей для админских интерфейсов.
// includeDeleted=false по умолчанию — мягко удалённые скрыты.
func (r *UserRepository) List(ctx context.Context, page, limit int, includeDeleted bool) ([]domain.User, int, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	offset := (page - 1) * limit

	where := "WHERE deleted_at IS NULL"
	if includeDeleted {
		where = ""
	}

	var total int
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM users `+where).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Pool.Query(ctx,
		`SELECT `+userColumns+` FROM users `+where+` ORDER BY id ASC LIMIT $1 OFFSET $2`,
		limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	users := make([]domain.User, 0, limit)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.Username, &u.AvatarURL, &u.Role,
			&u.IsBlocked, &u.BlockedAt, &u.DeletedAt, &u.TokenVersion, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// IncrementTokenVersion инвалидирует все ранее выпущенные access-токены пользователя.
// Используется при смене пароля и блокировке учётной записи.
func (r *UserRepository) IncrementTokenVersion(ctx context.Context, id int) error {
	query := `UPDATE users SET token_version = token_version + 1, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`
	tag, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}
