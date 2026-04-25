package service

import (
	"context"
	"errors"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/repository"
)

var (
	ErrInvalidRole      = errors.New("недопустимая роль")
	ErrCannotModifySelf = errors.New("нельзя выполнить действие над собственной учётной записью")
)

// AdminUserStore — контракт зависимости AdminService от хранилища пользователей.
// Шире, чем UserStore: нужен доступ к мягко удалённым и список с пагинацией.
type AdminUserStore interface {
	GetByIDIncludingDeleted(ctx context.Context, id int) (*domain.User, error)
	List(ctx context.Context, page, limit int, includeDeleted bool) ([]domain.User, int, error)
	SetBlocked(ctx context.Context, id int, blocked bool) error
	SoftDelete(ctx context.Context, id int) error
	SetRole(ctx context.Context, id int, role string) error
	IncrementTokenVersion(ctx context.Context, id int) error
}

// AdminService реализует функции модуля «Пользователи и доступ»:
// блокировка, удаление, смена роли, просмотр списка пользователей.
type AdminService struct {
	userRepo  AdminUserStore
	tokenRepo RefreshTokenStore
}

func NewAdminService(userRepo AdminUserStore, tokenRepo RefreshTokenStore) *AdminService {
	return &AdminService{userRepo: userRepo, tokenRepo: tokenRepo}
}

func (s *AdminService) ListUsers(ctx context.Context, page, limit int, includeDeleted bool) (*domain.AdminUsersListResponse, error) {
	users, total, err := s.userRepo.List(ctx, page, limit, includeDeleted)
	if err != nil {
		return nil, err
	}
	items := make([]domain.AdminUserBrief, 0, len(users))
	for i := range users {
		items = append(items, toAdminUserBrief(&users[i]))
	}
	if page < 1 {
		page = 1
	}
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	return &domain.AdminUsersListResponse{
		Users:      items,
		Pagination: domain.Pagination{Page: page, Limit: limit, Total: total},
	}, nil
}

func (s *AdminService) GetUser(ctx context.Context, id int) (*domain.AdminUserBrief, error) {
	u, err := s.userRepo.GetByIDIncludingDeleted(ctx, id)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, repository.ErrUserNotFound
	}
	b := toAdminUserBrief(u)
	return &b, nil
}

// SetBlocked блокирует/разблокирует пользователя.
// При блокировке инвалидирует все его сессии: удаляет refresh-токены и
// инкрементит token_version, чтобы старые access-токены тоже перестали работать.
func (s *AdminService) SetBlocked(ctx context.Context, actorID, targetID int, blocked bool) error {
	if actorID == targetID {
		return ErrCannotModifySelf
	}
	if err := s.userRepo.SetBlocked(ctx, targetID, blocked); err != nil {
		return err
	}
	if blocked {
		_ = s.tokenRepo.DeleteByUserID(ctx, targetID)
		_ = s.userRepo.IncrementTokenVersion(ctx, targetID)
	}
	return nil
}

// SoftDelete мягко удаляет учётную запись и инвалидирует refresh-токены.
func (s *AdminService) SoftDelete(ctx context.Context, actorID, targetID int) error {
	if actorID == targetID {
		return ErrCannotModifySelf
	}
	if err := s.userRepo.SoftDelete(ctx, targetID); err != nil {
		return err
	}
	_ = s.tokenRepo.DeleteByUserID(ctx, targetID)
	return nil
}

// SetRole меняет роль пользователя (user / moderator / admin).
func (s *AdminService) SetRole(ctx context.Context, actorID, targetID int, role string) error {
	if !domain.IsValidRole(role) {
		return ErrInvalidRole
	}
	if actorID == targetID {
		return ErrCannotModifySelf
	}
	if err := s.userRepo.SetRole(ctx, targetID, role); err != nil {
		return err
	}
	// Принудительно завершаем сессии и инвалидируем уже выпущенные access-токены,
	// чтобы новые запросы шли с актуальной ролью.
	_ = s.tokenRepo.DeleteByUserID(ctx, targetID)
	_ = s.userRepo.IncrementTokenVersion(ctx, targetID)
	return nil
}

func toAdminUserBrief(u *domain.User) domain.AdminUserBrief {
	return domain.AdminUserBrief{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
		Role:      u.Role,
		IsBlocked: u.IsBlocked,
		BlockedAt: u.BlockedAt,
		DeletedAt: u.DeletedAt,
		CreatedAt: u.CreatedAt,
	}
}
