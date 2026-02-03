package service

import (
	"context"
	"errors"
	"time"

	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/repository"
	"github.com/pro100kartochki/mozgoemka/pkg/jwt"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("неверный email или пароль")
	ErrEmailExists        = errors.New("пользователь с таким email уже существует")
	ErrRefreshTokenInvalid = errors.New("недействительный refresh token")
)

type AuthService struct {
	userRepo   *repository.UserRepository
	tokenRepo  *repository.RefreshTokenRepository
	jwtManager *jwt.Manager
}

func NewAuthService(userRepo *repository.UserRepository, tokenRepo *repository.RefreshTokenRepository, jwtManager *jwt.Manager) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		tokenRepo:  tokenRepo,
		jwtManager: jwtManager,
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

func (s *AuthService) issueTokens(ctx context.Context, u *domain.User) (*domain.TokenResponse, error) {
	access, err := s.jwtManager.GenerateAccessToken(u.ID, u.Email, u.Role)
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
