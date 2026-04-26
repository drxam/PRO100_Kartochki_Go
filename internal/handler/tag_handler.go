package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/service"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/validator"
)

type TagHandler struct {
	tagService *service.TagService
	validator  *validator.Validator
}

func NewTagHandler(tagService *service.TagService, v *validator.Validator) *TagHandler {
	return &TagHandler{tagService: tagService, validator: v}
}

func (h *TagHandler) List(c *gin.Context) {
	search := c.Query("search")
	list, err := h.tagService.ListWithSearch(c.Request.Context(), search)
	if err != nil {
		InternalError(c, "ошибка загрузки тегов")
		return
	}
	JSON(c, domain.TagsResponse{Tags: list})
}

func (h *TagHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	tag, err := h.tagService.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrTagNotFound {
			NotFound(c, err.Error())
			return
		}
		InternalError(c, "ошибка загрузки тега")
		return
	}
	JSON(c, tag)
}

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

func (h *TagHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	var req domain.UpdateTagRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	tag, err := h.tagService.Update(c.Request.Context(), id, req)
	if err != nil {
		if err == service.ErrTagNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrTagExists {
			Conflict(c, err.Error())
			return
		}
		InternalError(c, "ошибка обновления тега")
		return
	}
	JSON(c, tag)
}

func (h *TagHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	if err := h.tagService.Delete(c.Request.Context(), id); err != nil {
		if err == service.ErrTagNotFound {
			NotFound(c, err.Error())
			return
		}
		InternalError(c, "ошибка удаления тега")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Tag deleted successfully"})
}
