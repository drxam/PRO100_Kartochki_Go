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

type CardHandler struct {
	cardService *service.CardService
	validator   *validator.Validator
}

func NewCardHandler(cardService *service.CardService, v *validator.Validator) *CardHandler {
	return &CardHandler{cardService: cardService, validator: v}
}

// List возвращает все карточки пользователя (GET /api/cards)
func (h *CardHandler) List(c *gin.Context) {
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
	var categoryID, tagID *int
	if cid := c.Query("category_id"); cid != "" {
		if n, err := strconv.Atoi(cid); err == nil {
			categoryID = &n
		}
	}
	if tid := c.Query("tag_id"); tid != "" {
		if n, err := strconv.Atoi(tid); err == nil {
			tagID = &n
		}
	}
	search := c.Query("search")
	resp, err := h.cardService.ListByUserPaginated(c.Request.Context(), userID, page, limit, categoryID, tagID, search)
	if err != nil {
		InternalError(c, "ошибка загрузки карточек")
		return
	}
	JSON(c, resp)
}

// Create создаёт карточку (POST /api/cards — deck_id в body; или POST /api/decks/:deck_id/cards)
func (h *CardHandler) Create(c *gin.Context) {
	deckID := 0
	if idStr := c.Param("deck_id"); idStr != "" {
		if n, err := strconv.Atoi(idStr); err == nil {
			deckID = n
		}
	}
	userID := middleware.GetUserID(c)
	var req domain.CreateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestSimple(c, "неверный формат запроса")
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	if deckID == 0 && (req.DeckID == nil || *req.DeckID == 0) {
		BadRequestSimple(c, "deck_id обязателен")
		return
	}
	card, err := h.cardService.Create(c.Request.Context(), deckID, userID, req)
	if err != nil {
		if err == service.ErrCardForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка создания карточки")
		return
	}
	// Ответ в формате карточки с deck/category/tags
	item, _ := h.cardService.GetByIDForAPI(c.Request.Context(), card.ID, userID)
	if item != nil {
		c.JSON(http.StatusCreated, item)
		return
	}
	c.JSON(http.StatusCreated, card)
}

// GetByID возвращает карточку по ID (с deck, category, tags)
func (h *CardHandler) GetByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	userID := middleware.GetUserID(c)
	item, err := h.cardService.GetByIDForAPI(c.Request.Context(), id, userID)
	if err != nil {
		if err == service.ErrCardNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrCardForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка загрузки карточки")
		return
	}
	JSON(c, item)
}

// ListByDeck карточки набора (GET /api/decks/:deck_id/cards)
func (h *CardHandler) ListByDeck(c *gin.Context) {
	deckID, err := strconv.Atoi(c.Param("deck_id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID набора")
		return
	}
	userID := middleware.GetUserID(c)
	list, err := h.cardService.ListByDeck(c.Request.Context(), deckID, userID)
	if err != nil {
		if err == service.ErrCardForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка загрузки карточек")
		return
	}
	JSON(c, list)
}

// Update обновляет карточку
func (h *CardHandler) Update(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	userID := middleware.GetUserID(c)
	var req domain.UpdateCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequestSimple(c, "неверный формат запроса")
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}
	card, err := h.cardService.Update(c.Request.Context(), id, userID, req)
	if err != nil {
		if err == service.ErrCardNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrCardForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка обновления карточки")
		return
	}
	item, _ := h.cardService.GetByIDForAPI(c.Request.Context(), card.ID, userID)
	if item != nil {
		JSON(c, item)
		return
	}
	JSON(c, card)
}

// Delete удаляет карточку (200 + message)
func (h *CardHandler) Delete(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID")
		return
	}
	userID := middleware.GetUserID(c)
	if err := h.cardService.Delete(c.Request.Context(), id, userID); err != nil {
		if err == service.ErrCardNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrCardForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка удаления карточки")
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Card deleted successfully"})
}
