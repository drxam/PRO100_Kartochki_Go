package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/service"
	"github.com/pro100kartochki/mozgoemka/pkg/validator"
)

type TagHandler struct {
	tagService *service.TagService
	validator  *validator.Validator
}

func NewTagHandler(tagService *service.TagService, v *validator.Validator) *TagHandler {
	return &TagHandler{tagService: tagService, validator: v}
}

// List godoc
// @Summary      Список тегов
// @Tags         tags
// @Produce      json
// @Success      200   {array}   domain.Tag
// @Router       /tags [get]
func (h *TagHandler) List(c *gin.Context) {
	search := c.Query("search")
	list, err := h.tagService.ListWithSearch(c.Request.Context(), search)
	if err != nil {
		InternalError(c, "ошибка загрузки тегов")
		return
	}
	JSON(c, domain.TagsResponse{Tags: list})
}

// Create godoc
// @Summary      Создать тег
// @Tags         tags
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  domain.CreateTagRequest  true  "Название"
// @Success      201   {object}  domain.Tag
// @Failure      400   {object}  map[string]string
// @Router       /tags [post]
func (h *TagHandler) Create(c *gin.Context) {
	var req domain.CreateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	tag, err := h.tagService.Create(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrTagExists {
			Conflict(c, err.Error())
			return
		}
		InternalError(c, "ошибка создания тега")
		return
	}
	Created(c, tag)
}
