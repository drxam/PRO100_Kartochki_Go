package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/middleware"
	"github.com/drxam/PRO100_Kartochki_Go/internal/service"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/validator"
	"go.uber.org/zap"
)

type AuthHandler struct {
	authService     *service.AuthService
	validator       *validator.Validator
	devReturnReset  bool // dev-режим: возвращать reset_token прямо в ответе
}

func NewAuthHandler(authService *service.AuthService, v *validator.Validator) *AuthHandler {
	return &AuthHandler{authService: authService, validator: v}
}

// SetDevReturnResetToken включает выдачу reset_token в JSON-ответе на forgot-password.
// В production (письмо отправляется на email) должен быть false.
func (h *AuthHandler) SetDevReturnResetToken(enabled bool) {
	h.devReturnReset = enabled
}

// Register godoc
// @Summary  Регистрация нового пользователя
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body domain.RegisterRequest true "email + password"
// @Success  201 {object} domain.AuthRegisterResponse
// @Failure  400 {object} handler.ErrorPayload "ошибка валидации"
// @Failure  409 {object} handler.ErrorPayload "email уже существует"
// @Router   /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req domain.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	resp, err := h.authService.Register(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrEmailExists {
			Conflict(c, err.Error())
			return
		}
		InternalError(c, "ошибка регистрации")
		return
	}
	Audit(c, "auth.register", zap.Int("user_id", resp.User.ID), zap.String("email", req.Email))
	c.JSON(http.StatusCreated, resp)
}

// Login godoc
// @Summary  Вход в систему
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body domain.LoginRequest true "email + password"
// @Success  200 {object} domain.AuthLoginResponse
// @Failure  401 {object} handler.ErrorPayload "неверный email или пароль"
// @Failure  403 {object} handler.ErrorPayload "учётная запись заблокирована"
// @Router   /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	resp, err := h.authService.Login(c.Request.Context(), req)
	if err != nil {
		reason := "internal"
		switch err {
		case service.ErrInvalidCredentials:
			reason = "invalid_credentials"
			Unauthorized(c, err.Error())
		case service.ErrUserBlocked:
			reason = "blocked"
			Forbidden(c, err.Error())
		default:
			InternalError(c, "ошибка входа")
		}
		Audit(c, "auth.login.failure", zap.String("email", req.Email), zap.String("reason", reason))
		return
	}
	Audit(c, "auth.login.success", zap.Int("user_id", resp.User.ID), zap.String("email", req.Email))
	JSON(c, resp)
}

// Refresh godoc
// @Summary  Обновление access-токена по refresh-токену
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body domain.RefreshRequest true "refresh_token"
// @Success  200 {object} domain.AuthRefreshResponse
// @Failure  401 {object} handler.ErrorPayload "недействительный refresh-токен"
// @Failure  403 {object} handler.ErrorPayload "учётная запись заблокирована"
// @Router   /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req domain.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	resp, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch err {
		case service.ErrRefreshTokenInvalid:
			InvalidToken(c, err.Error())
		case service.ErrUserBlocked:
			Forbidden(c, err.Error())
		default:
			InternalError(c, "ошибка обновления токена")
		}
		return
	}
	JSON(c, resp)
}

// Logout godoc
// @Summary  Выход из системы (инвалидация refresh-токена)
// @Tags     auth
// @Security BearerAuth
// @Accept   json
// @Produce  json
// @Param    body body domain.RefreshRequest false "refresh_token (опционально)"
// @Success  200 {object} map[string]string
// @Router   /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	uid := middleware.GetUserID(c)
	var req domain.RefreshRequest
	// Опционально: можно передавать refresh_token в body для инвалидации
	if c.Request.ContentLength > 0 {
		_ = c.ShouldBindJSON(&req)
		if req.RefreshToken != "" {
			_ = h.authService.Logout(c.Request.Context(), req.RefreshToken)
		}
	}
	Audit(c, "auth.logout", zap.Int("user_id", uid))
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// ForgotPassword godoc
// @Summary  Запросить ссылку для сброса пароля
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body domain.ForgotPasswordRequest true "email"
// @Success  200 {object} domain.ForgotPasswordResponse
// @Failure  400 {object} handler.ErrorPayload
// @Router   /auth/forgot-password [post]
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req domain.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}

	token, ok, err := h.authService.RequestPasswordReset(c.Request.Context(), req.Email)
	if err != nil {
		InternalError(c, "ошибка запроса сброса пароля")
		return
	}
	Audit(c, "auth.password_reset.requested", zap.String("email", req.Email), zap.Bool("issued", ok))

	// Одинаковый ответ независимо от того, существует ли email — чтобы не выдавать
	// перечисление учёток (account enumeration).
	resp := domain.ForgotPasswordResponse{
		Message: "Если такой email зарегистрирован, на него отправлена ссылка для сброса пароля",
	}
	if ok && h.devReturnReset {
		resp.ResetToken = token
	}
	c.JSON(http.StatusOK, resp)
}

// ResetPassword godoc
// @Summary  Сменить пароль по токену из ссылки
// @Tags     auth
// @Accept   json
// @Produce  json
// @Param    body body domain.ResetPasswordRequest true "token + new_password"
// @Success  200 {object} map[string]string
// @Failure  400 {object} handler.ErrorPayload "недействительный/истёкший токен"
// @Failure  403 {object} handler.ErrorPayload "учётная запись заблокирована"
// @Router   /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req domain.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		reason := "internal"
		switch err {
		case service.ErrInvalidResetToken:
			reason = "invalid_token"
			BadRequestSimple(c, err.Error())
		case service.ErrResetTokenExpired:
			reason = "expired"
			BadRequestSimple(c, err.Error())
		case service.ErrUserBlocked:
			reason = "blocked"
			Forbidden(c, err.Error())
		default:
			InternalError(c, "ошибка сброса пароля")
		}
		Audit(c, "auth.password_reset.failure", zap.String("reason", reason))
		return
	}
	Audit(c, "auth.password_reset.applied")
	c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменён"})
}
