package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/middleware"
	"github.com/drxam/PRO100_Kartochki_Go/internal/service"
)

type FavoriteHandler struct {
	favoriteService *service.FavoriteService
}

func NewFavoriteHandler(favoriteService *service.FavoriteService) *FavoriteHandler {
	return &FavoriteHandler{favoriteService: favoriteService}
}

// Add — POST /api/decks/:id/favorite
func (h *FavoriteHandler) Add(c *gin.Context) {
	deckID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID набора")
		return
	}
	userID := middleware.GetUserID(c)

	if err := h.favoriteService.Add(c.Request.Context(), userID, deckID); err != nil {
		switch err {
		case service.ErrDeckNotFound:
			NotFound(c, err.Error())
		case service.ErrDeckForbidden:
			Forbidden(c, err.Error())
		case service.ErrAlreadyFavorite:
			Conflict(c, err.Error())
		default:
			InternalError(c, "ошибка добавления в избранное")
		}
		return
	}
	c.JSON(http.StatusOK, domain.FavoriteResponse{
		DeckID:     deckID,
		IsFavorite: true,
		Message:    "Набор добавлен в избранное",
	})
}

// Remove — DELETE /api/decks/:id/favorite
func (h *FavoriteHandler) Remove(c *gin.Context) {
	deckID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID набора")
		return
	}
	userID := middleware.GetUserID(c)

	if err := h.favoriteService.Remove(c.Request.Context(), userID, deckID); err != nil {
		if err == service.ErrNotFavorite {
			NotFound(c, err.Error())
			return
		}
		InternalError(c, "ошибка удаления из избранного")
		return
	}
	c.JSON(http.StatusOK, domain.FavoriteResponse{
		DeckID:     deckID,
		IsFavorite: false,
		Message:    "Набор убран из избранного",
	})
}

// List — GET /api/favorites
func (h *FavoriteHandler) List(c *gin.Context) {
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

	resp, err := h.favoriteService.List(c.Request.Context(), userID, page, limit)
	if err != nil {
		InternalError(c, "ошибка загрузки избранного")
		return
	}
	JSON(c, resp)
}
