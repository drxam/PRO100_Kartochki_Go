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
	authService *service.AuthService
	validator   *validator.Validator
}

func NewAuthHandler(authService *service.AuthService, v *validator.Validator) *AuthHandler {
	return &AuthHandler{authService: authService, validator: v}
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
		if err == service.ErrInvalidCredentials {
			Unauthorized(c, err.Error())
			return
		}
		InternalError(c, "ошибка входа")
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
		if err == service.ErrRefreshTokenInvalid {
			InvalidToken(c, err.Error())
			return
		}
		InternalError(c, "ошибка обновления токена")
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
	// Заглушка: в реальности отправить письмо со ссылкой на сброс пароля
	_ = req
	c.JSON(http.StatusOK, gin.H{"message": "Password reset link sent to email"})
}
