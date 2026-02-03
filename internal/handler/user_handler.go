package handler

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/middleware"
	"github.com/pro100kartochki/mozgoemka/internal/service"
	"github.com/pro100kartochki/mozgoemka/pkg/validator"
)

const maxAvatarSize = 5 << 20 // 5MB

type UserHandler struct {
	userService *service.UserService
	validator   *validator.Validator
}

func NewUserHandler(userService *service.UserService, v *validator.Validator) *UserHandler {
	return &UserHandler{userService: userService, validator: v}
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	resp, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil || resp == nil {
		NotFound(c, "пользователь не найден")
		return
	}
	JSON(c, resp)
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req domain.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	u, err := h.userService.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		InternalError(c, "ошибка обновления профиля")
		return
	}
	// Ответ без password_hash
	u.PasswordHash = ""
	JSON(c, u)
}

func (h *UserHandler) UploadAvatar(c *gin.Context) {
	userID := middleware.GetUserID(c)
	file, err := c.FormFile("avatar")
	if err != nil {
		BadRequest(c, "требуется файл avatar (JPG/PNG, макс. 5MB)", nil)
		return
	}
	if file.Size > maxAvatarSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "файл слишком большой (макс. 5MB)"})
		return
	}
	ext := file.Filename
	if len(ext) > 4 {
		ext = ext[len(ext)-4:]
	}
	if ext != ".jpg" && ext != "jpeg" && ext != ".png" && ext != ".PNG" && ext != ".JPG" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "допустимы только JPG и PNG"})
		return
	}
	src, err := file.Open()
	if err != nil {
		InternalError(c, "ошибка чтения файла")
		return
	}
	defer src.Close()
	data, err := io.ReadAll(src)
	if err != nil {
		InternalError(c, "ошибка чтения файла")
		return
	}
	avatarURL, err := h.userService.UploadAvatar(c.Request.Context(), userID, file.Filename, data)
	if err != nil {
		InternalError(c, "ошибка загрузки аватара")
		return
	}
	c.JSON(http.StatusOK, gin.H{"avatar_url": avatarURL})
}
