package main

import (
	"context"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// bootstrapAdmin при старте создаёт пользователя-администратора либо повышает
// существующую учётную запись до роли admin. Идемпотентно: при повторном запуске
// с тем же email ничего не меняется, если пользователь уже admin.
//
// Не возвращает фатальную ошибку — если что-то пошло не так, логирует и
// сервер продолжает работу: bootstrap полезен, но не должен ломать запуск.
func bootstrapAdmin(ctx context.Context, logger *zap.Logger, users *repository.UserRepository, email, password string) {
	if email == "" || password == "" {
		return
	}
	logger = logger.With(zap.String("bootstrap_email", email))

	existing, err := users.GetByEmail(ctx, email)
	if err != nil {
		logger.Error("bootstrap_admin: lookup failed", zap.Error(err))
		return
	}

	if existing == nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			logger.Error("bootstrap_admin: hash failed", zap.Error(err))
			return
		}
		u := &domain.User{
			Email:        email,
			PasswordHash: string(hash),
			Role:         string(domain.RoleAdmin),
		}
		if err := users.Create(ctx, u); err != nil {
			logger.Error("bootstrap_admin: create failed", zap.Error(err))
			return
		}
		logger.Info("bootstrap_admin: created", zap.Int("user_id", u.ID))
		return
	}

	if existing.Role == string(domain.RoleAdmin) {
		logger.Info("bootstrap_admin: already admin", zap.Int("user_id", existing.ID))
		return
	}

	if err := users.SetRole(ctx, existing.ID, string(domain.RoleAdmin)); err != nil {
		logger.Error("bootstrap_admin: promote failed", zap.Error(err))
		return
	}
	// После повышения роли нужно инвалидировать ранее выданные access-токены.
	_ = users.IncrementTokenVersion(ctx, existing.ID)
	logger.Info("bootstrap_admin: promoted to admin", zap.Int("user_id", existing.ID))
}
