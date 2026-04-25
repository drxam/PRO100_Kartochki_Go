package main

import (
	"context"
	"time"

	"github.com/drxam/PRO100_Kartochki_Go/internal/repository"
	"go.uber.org/zap"
)

// usedTokenRetention — сколько держим использованный токен сброса пароля
// перед удалением (короткое окно для аудита).
const usedTokenRetention = 7 * 24 * time.Hour

// runCleanupLoop запускает периодическую чистку истёкших refresh-токенов и
// токенов восстановления пароля. Возвращается, когда ctx отменён.
//
// Один проход выполняется сразу при старте, далее по тикеру.
func runCleanupLoop(
	ctx context.Context,
	logger *zap.Logger,
	refreshRepo *repository.RefreshTokenRepository,
	resetRepo *repository.PasswordResetRepository,
	interval time.Duration,
) {
	cleanupOnce(ctx, logger, refreshRepo, resetRepo)

	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			cleanupOnce(ctx, logger, refreshRepo, resetRepo)
		}
	}
}

func cleanupOnce(
	ctx context.Context,
	logger *zap.Logger,
	refreshRepo *repository.RefreshTokenRepository,
	resetRepo *repository.PasswordResetRepository,
) {
	if n, err := refreshRepo.DeleteExpired(ctx); err != nil {
		logger.Warn("cleanup: refresh_tokens failed", zap.Error(err))
	} else if n > 0 {
		logger.Info("cleanup: refresh_tokens deleted", zap.Int64("rows", n))
	}

	if n, err := resetRepo.DeleteExpired(ctx, usedTokenRetention); err != nil {
		logger.Warn("cleanup: password_reset_tokens failed", zap.Error(err))
	} else if n > 0 {
		logger.Info("cleanup: password_reset_tokens deleted", zap.Int64("rows", n))
	}
}
