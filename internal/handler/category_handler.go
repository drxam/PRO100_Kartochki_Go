package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/service"
	"github.com/pro100kartochki/mozgoemka/pkg/validator"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
	validator       *validator.Validator
}

func NewCategoryHandler(categoryService *service.CategoryService, v *validator.Validator) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService, validator: v}
}

// List godoc
// @Summary      Список категорий
// @Tags         categories
// @Produce      json
// @Success      200   {array}   domain.Category
// @Router       /categories [get]
func (h *CategoryHandler) List(c *gin.Context) {
	list, err := h.categoryService.List(c.Request.Context())
	if err != nil {
		InternalError(c, "ошибка загрузки категорий")
		return
	}
	JSON(c, domain.CategoriesResponse{Categories: list})
}

// Create godoc
// @Summary      Создать категорию
// @Tags         categories
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  domain.CreateCategoryRequest  true  "Название"
// @Success      201   {object}  domain.Category
// @Failure      400   {object}  map[string]string
// @Router       /categories [post]
func (h *CategoryHandler) Create(c *gin.Context) {
	var req domain.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	cat, err := h.categoryService.Create(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrCategoryExists {
			Conflict(c, err.Error())
			return
		}
		InternalError(c, "ошибка создания категории")
		return
	}
	Created(c, cat)
}
