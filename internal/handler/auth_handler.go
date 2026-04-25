package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/middleware"
	"github.com/pro100kartochki/mozgoemka/internal/service"
	"github.com/pro100kartochki/mozgoemka/pkg/validator"
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
	c.JSON(http.StatusCreated, resp)
}

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
		switch err {
		case service.ErrInvalidCredentials:
			Unauthorized(c, err.Error())
		case service.ErrUserBlocked:
			Forbidden(c, err.Error())
		default:
			InternalError(c, "ошибка входа")
		}
		return
	}
	JSON(c, resp)
}

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

func (h *AuthHandler) Logout(c *gin.Context) {
	_ = middleware.GetUserID(c)
	var req domain.RefreshRequest
	// Опционально: можно передавать refresh_token в body для инвалидации
	if c.Request.ContentLength > 0 {
		_ = c.ShouldBindJSON(&req)
		if req.RefreshToken != "" {
			_ = h.authService.Logout(c.Request.Context(), req.RefreshToken)
		}
	}
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
		switch err {
		case service.ErrInvalidResetToken:
			BadRequestSimple(c, err.Error())
		case service.ErrResetTokenExpired:
			BadRequestSimple(c, err.Error())
		case service.ErrUserBlocked:
			Forbidden(c, err.Error())
		default:
			InternalError(c, "ошибка сброса пароля")
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Пароль успешно изменён"})
}
