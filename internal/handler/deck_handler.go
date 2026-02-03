package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/middleware"
	"github.com/pro100kartochki/mozgoemka/internal/service"
	"github.com/pro100kartochki/mozgoemka/pkg/validator"
)

type DeckHandler struct {
	deckService *service.DeckService
	validator   *validator.Validator
}

func NewDeckHandler(deckService *service.DeckService, v *validator.Validator) *DeckHandler {
	return &DeckHandler{deckService: deckService, validator: v}
}

func (h *DeckHandler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req domain.CreateDeckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	deck, err := h.deckService.Create(c.Request.Context(), userID, req)
	if err != nil {
		InternalError(c, "ошибка создания набора")
		return
	}
	c.JSON(http.StatusCreated, deck)
}

func (h *DeckHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "неверный ID", nil)
		return
	}
	userID := middleware.GetUserID(c)
	deck, err := h.deckService.GetByID(c.Request.Context(), id, userID)
	if err != nil {
		if err == service.ErrDeckNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrDeckForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка загрузки набора")
		return
	}
	JSON(c, deck)
}

func (h *DeckHandler) ListMine(c *gin.Context) {
	userID := middleware.GetUserID(c)
	page, limit := 1, 20
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}
	var categoryID *int
	if cid := c.Query("category_id"); cid != "" {
		if n, err := strconv.Atoi(cid); err == nil {
			categoryID = &n
		}
	}
	search := c.Query("search")
	resp, err := h.deckService.ListByUserPaginated(c.Request.Context(), userID, page, limit, categoryID, search)
	if err != nil {
		InternalError(c, "ошибка загрузки наборов")
		return
	}
	JSON(c, resp)
}

func (h *DeckHandler) ListPublic(c *gin.Context) {
	limit := 20
	offset := 0
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if o := c.Query("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}
	list, err := h.deckService.ListPublic(c.Request.Context(), limit, offset)
	if err != nil {
		InternalError(c, "ошибка загрузки наборов")
		return
	}
	JSON(c, list)
}

func (h *DeckHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "неверный ID", nil)
		return
	}
	userID := middleware.GetUserID(c)
	var req domain.UpdateDeckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	deck, err := h.deckService.Update(c.Request.Context(), id, userID, req)
	if err != nil {
		if err == service.ErrDeckNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrDeckForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка обновления набора")
		return
	}
	JSON(c, deck)
}

func (h *DeckHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequest(c, "неверный ID", nil)
		return
	}
	userID := middleware.GetUserID(c)
	if err := h.deckService.Delete(c.Request.Context(), id, userID); err != nil {
		if err == service.ErrDeckNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrDeckForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка удаления набора")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deck deleted successfully"})
}

// ListPublicPaginated — GET /api/public/decks (без авторизации)
func (h *DeckHandler) ListPublicPaginated(c *gin.Context) {
	page, limit := 1, 20
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 100 {
		limit = 100
	}
	var categoryID *int
	if cid := c.Query("category_id"); cid != "" {
		if n, err := strconv.Atoi(cid); err == nil {
			categoryID = &n
		}
	}
	search := c.Query("search")
	sortBy := c.DefaultQuery("sort_by", "recent") // popular, recent, cards_count
	resp, err := h.deckService.ListPublicPaginated(c.Request.Context(), page, limit, categoryID, search, sortBy)
	if err != nil {
		InternalError(c, "ошибка загрузки наборов")
		return
	}
	JSON(c, resp)
}

// GetPublicByID — GET /api/public/decks/:id (без авторизации)
func (h *DeckHandler) GetPublicByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	deck, err := h.deckService.GetPublicByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrDeckNotFound {
			NotFound(c, err.Error())
			return
		}
		InternalError(c, "ошибка загрузки набора")
		return
	}
	JSON(c, deck)
}
