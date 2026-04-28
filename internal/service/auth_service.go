package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/mailer"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

// Контракты зависимостей AuthService — позволяют подменять репозитории мок-объектами в unit-тестах.

type UserStore interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id int) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	UpdatePassword(ctx context.Context, id int, passwordHash string) error
	IncrementTokenVersion(ctx context.Context, id int) error
}

type RefreshTokenStore interface {
	Create(ctx context.Context, rt *domain.RefreshToken) error
	GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error)
	DeleteByToken(ctx context.Context, token string) error
	DeleteByUserID(ctx context.Context, userID int) error
}

type PasswordResetStore interface {
	Create(ctx context.Context, prt *domain.PasswordResetToken) error
	GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id int) error
	InvalidateActiveForUser(ctx context.Context, userID int) error
}

// JWTManager — минимальный контракт для подмены jwt.Manager (если потребуется);
// в продакшене используется *jwt.Manager.
type JWTManager interface {
	GenerateAccessToken(userID int, email, role string, tokenVersion int) (string, error)
	GenerateRefreshToken(userID int) (string, time.Time, error)
	ParseAccessToken(token string) (*jwt.Claims, error)
	ParseRefreshToken(token string) (*jwt.Claims, error)
}

var (
	ErrInvalidCredentials  = errors.New("неверный email или пароль")
	ErrEmailExists         = errors.New("пользователь с таким email уже существует")
	ErrRefreshTokenInvalid = errors.New("недействительный refresh token")
	ErrUserBlocked         = errors.New("учётная запись заблокирована")
	ErrInvalidResetToken   = errors.New("недействительный токен сброса пароля")
	ErrResetTokenExpired   = errors.New("срок действия ссылки истёк")
)

const passwordResetTTL = 1 * time.Hour

type AuthService struct {
	userRepo     UserStore
	tokenRepo    RefreshTokenStore
	resetRepo    PasswordResetStore
	jwtManager   JWTManager
	mailer       mailer.Mailer
	appPublicURL string // куда ведут ссылки в письмах
}

func NewAuthService(
	userRepo UserStore,
	tokenRepo RefreshTokenStore,
	resetRepo PasswordResetStore,
	jwtManager JWTManager,
	mailerSvc mailer.Mailer,
	appPublicURL string,
) *AuthService {
	if mailerSvc == nil {
		mailerSvc = &mailer.NoopMailer{}
	}
	return &AuthService{
		userRepo:     userRepo,
		tokenRepo:    tokenRepo,
		resetRepo:    resetRepo,
		jwtManager:   jwtManager,
		mailer:       mailerSvc,
		appPublicURL: appPublicURL,
	}
}

func (s *AuthService) Register(ctx context.Context, req domain.RegisterRequest) (*domain.AuthRegisterResponse, error) {
	existing, _ := s.userRepo.GetByEmail(ctx, req.Email)
	if existing != nil {
		return nil, ErrEmailExists
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         string(domain.RoleUser),
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, err
	}
	return &domain.AuthRegisterResponse{
		User:         domain.AuthUserBrief{ID: u.ID, Email: u.Email, Role: u.Role},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *AuthService) Login(ctx context.Context, req domain.LoginRequest) (*domain.AuthLoginResponse, error) {
	u, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil || u == nil {
		return nil, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}
	if u.IsBlocked {
		return nil, ErrUserBlocked
	}
	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, err
	}
	return &domain.AuthLoginResponse{
		User:         domain.AuthUserFull{ID: u.ID, Email: u.Email, Username: u.Username, AvatarURL: u.AvatarURL, Role: u.Role},
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*domain.AuthRefreshResponse, error) {
	claims, err := s.jwtManager.ParseRefreshToken(refreshToken)
	if err != nil {
		return nil, ErrRefreshTokenInvalid
	}
	stored, err := s.tokenRepo.GetByToken(ctx, refreshToken)
	if err != nil || stored == nil || stored.UserID != claims.UserID || time.Now().After(stored.ExpiresAt) {
		return nil, ErrRefreshTokenInvalid
	}
	u, err := s.userRepo.GetByID(ctx, claims.UserID)
	if err != nil || u == nil {
		return nil, ErrRefreshTokenInvalid
	}
	if u.IsBlocked {
		_ = s.tokenRepo.DeleteByUserID(ctx, u.ID)
		return nil, ErrUserBlocked
	}
	_ = s.tokenRepo.DeleteByToken(ctx, refreshToken)
	tokens, err := s.issueTokens(ctx, u)
	if err != nil {
		return nil, err
	}
	return &domain.AuthRefreshResponse{AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.tokenRepo.DeleteByToken(ctx, refreshToken)
}

// RequestPasswordReset выпускает одноразовый токен сброса пароля.
//
// Возвращает (token, ok, err): ok=false означает «email не зарегистрирован
// или учётка заблокирована» — handler в обоих случаях должен отвечать клиенту
// одинаково (см. ТЗ §4.2 «Восстановление доступа»), чтобы не выдавать факт
// существования email. err — настоящая внутренняя ошибка (БД и т. п.).
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (token string, ok bool, err error) {
	u, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", false, err
	}
	if u == nil || u.IsBlocked {
		return "", false, nil
	}

	raw, err := generateResetToken()
	if err != nil {
		return "", false, err
	}

	// Любые ранее выпущенные активные токены гасим — действующая ссылка только одна.
	_ = s.resetRepo.InvalidateActiveForUser(ctx, u.ID)

	prt := &domain.PasswordResetToken{
		UserID:    u.ID,
		Token:     raw,
		ExpiresAt: time.Now().Add(passwordResetTTL),
	}
	if err := s.resetRepo.Create(ctx, prt); err != nil {
		return "", false, err
	}

	// Отправляем письмо со ссылкой. Ошибка отправки НЕ выдаётся клиенту —
	// это могло бы выдать существование email (account enumeration);
	// mailer сам логирует проблему.
	body := mailer.PasswordResetHTML(s.appPublicURL, raw)
	_ = s.mailer.Send(ctx, email, mailer.PasswordResetSubject, body)

	return raw, true, nil
}

// ResetPassword применяет токен: меняет пароль пользователя, помечает токен
// использованным и сбрасывает все его refresh-сессии.
func (s *AuthService) ResetPassword(ctx context.Context, token, newPassword string) error {
	prt, err := s.resetRepo.GetByToken(ctx, token)
	if err != nil {
		return err
	}
	if prt == nil || prt.UsedAt != nil {
		return ErrInvalidResetToken
	}
	if time.Now().After(prt.ExpiresAt) {
		return ErrResetTokenExpired
	}

	u, err := s.userRepo.GetByID(ctx, prt.UserID)
	if err != nil {
		return err
	}
	if u == nil {
		return ErrInvalidResetToken
	}
	if u.IsBlocked {
		return ErrUserBlocked
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := s.userRepo.UpdatePassword(ctx, u.ID, string(hash)); err != nil {
		return err
	}
	if err := s.resetRepo.MarkUsed(ctx, prt.ID); err != nil {
		return err
	}
	// Все активные сессии пользователя должны прерваться: refresh-токены удаляем,
	// а уже выпущенные access-токены инвалидируем повышением token_version.
	_ = s.tokenRepo.DeleteByUserID(ctx, u.ID)
	_ = s.userRepo.IncrementTokenVersion(ctx, u.ID)
	return nil
}

func generateResetToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *AuthService) issueTokens(ctx context.Context, u *domain.User) (*domain.TokenResponse, error) {
	access, err := s.jwtManager.GenerateAccessToken(u.ID, u.Email, u.Role, u.TokenVersion)
	if err != nil {
		return nil, err
	}
	refresh, expiresAt, err := s.jwtManager.GenerateRefreshToken(u.ID)
	if err != nil {
		return nil, err
	}
	rt := &domain.RefreshToken{UserID: u.ID, Token: refresh, ExpiresAt: expiresAt}
	if err := s.tokenRepo.Create(ctx, rt); err != nil {
		return nil, err
	}
	return &domain.TokenResponse{AccessToken: access, RefreshToken: refresh}, nil
}
