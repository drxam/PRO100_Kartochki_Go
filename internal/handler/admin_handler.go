package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/middleware"
	"github.com/pro100kartochki/mozgoemka/internal/repository"
	"github.com/pro100kartochki/mozgoemka/internal/service"
	"github.com/pro100kartochki/mozgoemka/pkg/validator"
	"go.uber.org/zap"
)

// AdminHandler — эндпоинты управления учётными записями (модуль «Пользователи и доступ»).
type AdminHandler struct {
	adminService *service.AdminService
	validator    *validator.Validator
}

func NewAdminHandler(adminService *service.AdminService, v *validator.Validator) *AdminHandler {
	return &AdminHandler{adminService: adminService, validator: v}
}

// ListUsers godoc
// @Summary  Список пользователей
// @Tags     admin
// @Security BearerAuth
// @Param    page             query int  false "Номер страницы (по умолчанию 1)"
// @Param    limit            query int  false "Размер страницы (по умолчанию 20)"
// @Param    include_deleted  query bool false "Показывать ли мягко удалённых"
// @Success  200 {object} domain.AdminUsersListResponse
// @Failure  401 {object} handler.ErrorPayload
// @Failure  403 {object} handler.ErrorPayload
// @Router   /admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	includeDeleted := c.DefaultQuery("include_deleted", "false") == "true"
	resp, err := h.adminService.ListUsers(c.Request.Context(), page, limit, includeDeleted)
	if err != nil {
		InternalError(c, "ошибка получения списка пользователей")
		return
	}
	JSON(c, resp)
}

// GetUser godoc
// @Summary  Получить пользователя по id
// @Tags     admin
// @Security BearerAuth
// @Param    id path int true "ID пользователя"
// @Success  200 {object} domain.AdminUserBrief
// @Failure  404 {object} handler.ErrorPayload
// @Router   /admin/users/{id} [get]
func (h *AdminHandler) GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		BadRequestSimple(c, "неверный id")
		return
	}
	u, err := h.adminService.GetUser(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrUserNotFound {
			NotFound(c, "пользователь не найден")
			return
		}
		InternalError(c, "ошибка получения пользователя")
		return
	}
	JSON(c, u)
}

// BlockUser godoc
// @Summary  Заблокировать или разблокировать пользователя
// @Tags     admin
// @Security BearerAuth
// @Param    id   path int                            true "ID пользователя"
// @Param    body body domain.AdminBlockUserRequest   true "blocked: true|false"
// @Success  204 "No Content"
// @Failure  403 {object} handler.ErrorPayload
// @Failure  404 {object} handler.ErrorPayload
// @Router   /admin/users/{id}/block [patch]
func (h *AdminHandler) BlockUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		BadRequestSimple(c, "неверный id")
		return
	}
	var req domain.AdminBlockUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	actorID := middleware.GetUserID(c)
	if err := h.adminService.SetBlocked(c.Request.Context(), actorID, id, req.Blocked); err != nil {
		switch err {
		case service.ErrCannotModifySelf:
			Forbidden(c, err.Error())
		case repository.ErrUserNotFound:
			NotFound(c, "пользователь не найден")
		default:
			InternalError(c, "ошибка изменения статуса блокировки")
		}
		return
	}
	Audit(c, "admin.user.blocked",
		zap.Int("actor_id", actorID),
		zap.Int("target_id", id),
		zap.Bool("blocked", req.Blocked),
	)
	NoContent(c)
}

// DeleteUser godoc
// @Summary  Мягко удалить пользователя
// @Tags     admin
// @Security BearerAuth
// @Param    id path int true "ID пользователя"
// @Success  204 "No Content"
// @Failure  403 {object} handler.ErrorPayload
// @Failure  404 {object} handler.ErrorPayload
// @Router   /admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		BadRequestSimple(c, "неверный id")
		return
	}
	actorID := middleware.GetUserID(c)
	if err := h.adminService.SoftDelete(c.Request.Context(), actorID, id); err != nil {
		switch err {
		case service.ErrCannotModifySelf:
			Forbidden(c, err.Error())
		case repository.ErrUserNotFound:
			NotFound(c, "пользователь не найден")
		default:
			InternalError(c, "ошибка удаления пользователя")
		}
		return
	}
	Audit(c, "admin.user.deleted",
		zap.Int("actor_id", actorID),
		zap.Int("target_id", id),
	)
	NoContent(c)
}

// SetUserRole godoc
// @Summary  Назначить роль пользователю
// @Tags     admin
// @Security BearerAuth
// @Param    id   path int                          true "ID пользователя"
// @Param    body body domain.AdminSetRoleRequest   true "role: user|moderator|admin"
// @Success  204 "No Content"
// @Failure  400 {object} handler.ErrorPayload
// @Failure  403 {object} handler.ErrorPayload
// @Router   /admin/users/{id}/role [patch]
func (h *AdminHandler) SetUserRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		BadRequestSimple(c, "неверный id")
		return
	}
	var req domain.AdminSetRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	actorID := middleware.GetUserID(c)
	if err := h.adminService.SetRole(c.Request.Context(), actorID, id, req.Role); err != nil {
		switch err {
		case service.ErrInvalidRole:
			BadRequestSimple(c, err.Error())
		case service.ErrCannotModifySelf:
			Forbidden(c, err.Error())
		case repository.ErrUserNotFound:
			NotFound(c, "пользователь не найден")
		default:
			InternalError(c, "ошибка изменения роли")
		}
		return
	}
	Audit(c, "admin.user.role_changed",
		zap.Int("actor_id", actorID),
		zap.Int("target_id", id),
		zap.String("role", req.Role),
	)
	NoContent(c)
}
