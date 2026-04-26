package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/service"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/validator"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
	validator       *validator.Validator
}

func NewCategoryHandler(categoryService *service.CategoryService, v *validator.Validator) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService, validator: v}
}

func (h *CategoryHandler) List(c *gin.Context) {
	list, err := h.categoryService.List(c.Request.Context())
	if err != nil {
		InternalError(c, "ошибка загрузки категорий")
		return
	}
	JSON(c, domain.CategoriesResponse{Categories: list})
}

func (h *CategoryHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	cat, err := h.categoryService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrCategoryNotFound {
			NotFound(c, err.Error())
			return
		}
		InternalError(c, "ошибка загрузки категории")
		return
	}
	JSON(c, cat)
}

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

func (h *CategoryHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	var req domain.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	cat, err := h.categoryService.Update(c.Request.Context(), id, req)
	if err != nil {
		if err == service.ErrCategoryNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrCategoryExists {
			Conflict(c, err.Error())
			return
		}
		InternalError(c, "ошибка обновления категории")
		return
	}
	JSON(c, cat)
}

func (h *CategoryHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	if err := h.categoryService.Delete(c.Request.Context(), id); err != nil {
		if err == service.ErrCategoryNotFound {
			NotFound(c, err.Error())
			return
		}
		InternalError(c, "ошибка удаления категории")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}
