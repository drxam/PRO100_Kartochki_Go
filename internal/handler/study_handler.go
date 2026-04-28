package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/middleware"
	"github.com/drxam/PRO100_Kartochki_Go/internal/service"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/validator"
)

type StudyHandler struct {
	studyService *service.StudyService
	validator    *validator.Validator
}

func NewStudyHandler(studyService *service.StudyService, v *validator.Validator) *StudyHandler {
	return &StudyHandler{studyService: studyService, validator: v}
}

// StartSession — POST /api/decks/:id/study/start
func (h *StudyHandler) StartSession(c *gin.Context) {
	deckID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID набора")
		return
	}
	userID := middleware.GetUserID(c)

	session, card, err := h.studyService.StartSession(c.Request.Context(), userID, deckID)
	if err != nil {
		switch err {
		case service.ErrDeckNotFound:
			NotFound(c, err.Error())
		case service.ErrDeckForbidden:
			Forbidden(c, err.Error())
		case service.ErrNothingToReview:
			c.JSON(http.StatusOK, gin.H{
				"message":     "Все карточки изучены! Следующее повторение запланировано.",
				"total_cards": 0,
			})
		default:
			InternalError(c, "ошибка запуска сессии")
		}
		return
	}

	resp := domain.StartStudyResponse{
		SessionID:  session.ID,
		TotalCards: session.CardsTotal,
		Status:     session.Status,
	}
	if card != nil {
		resp.Card = &domain.StudyCard{
			ID:       card.ID,
			Question: card.Question,
			Answer:   card.Answer,
		}
	}
	c.JSON(http.StatusCreated, resp)
}

// ReviewCard — POST /api/study/sessions/:id/review
func (h *StudyHandler) ReviewCard(c *gin.Context) {
	sessionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID сессии")
		return
	}
	userID := middleware.GetUserID(c)

	var req domain.ReviewCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "неверный формат запроса", nil)
		return
	}
	if errs := h.validator.Validate(&req); errs != nil {
		BadRequest(c, "ошибка валидации", errs)
		return
	}

	session, nextCard, err := h.studyService.ReviewCard(c.Request.Context(), userID, sessionID, req.CardID, *req.Quality)
	if err != nil {
		switch err {
		case service.ErrSessionNotFound:
			NotFound(c, err.Error())
		case service.ErrSessionForbidden:
			Forbidden(c, err.Error())
		case service.ErrSessionCompleted:
			BadRequestSimple(c, err.Error())
		case service.ErrCardNotFound:
			NotFound(c, err.Error())
		default:
			InternalError(c, "ошибка обработки ответа")
		}
		return
	}

	resp := domain.ReviewCardResponse{
		Progress: domain.CardProgressDTO{
			CardID:      req.CardID,
			NextReviewAt: time.Now().Format(time.RFC3339),
		},
	}

	if nextCard != nil {
		resp.NextCard = &domain.StudyCard{
			ID:       nextCard.ID,
			Question: nextCard.Question,
			Answer:   nextCard.Answer,
		}
	} else {
		// Сессия завершена
		resp.Summary = &domain.SessionSummary{
			SessionID:     session.ID,
			CardsReviewed: session.CardsReviewed,
			CardsCorrect:  session.CardsCorrect,
			AccuracyPct:   service.SessionAccuracyPct(session),
			Duration:      service.SessionDuration(session),
		}
	}

	JSON(c, resp)
}

// GetSession — GET /api/study/sessions/:id
func (h *StudyHandler) GetSession(c *gin.Context) {
	sessionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID сессии")
		return
	}
	userID := middleware.GetUserID(c)

	session, err := h.studyService.GetSessionByID(c.Request.Context(), userID, sessionID)
	if err != nil {
		if err == service.ErrSessionNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrSessionForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка загрузки сессии")
		return
	}
	JSON(c, sessionToDTO(session))
}

// FinishSession — POST /api/study/sessions/:id/finish
func (h *StudyHandler) FinishSession(c *gin.Context) {
	sessionID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID сессии")
		return
	}
	userID := middleware.GetUserID(c)

	session, err := h.studyService.FinishSession(c.Request.Context(), userID, sessionID)
	if err != nil {
		if err == service.ErrSessionNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrSessionForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка завершения сессии")
		return
	}
	JSON(c, gin.H{
		"session": sessionToDTO(session),
		"summary": domain.SessionSummary{
			SessionID:     session.ID,
			CardsReviewed: session.CardsReviewed,
			CardsCorrect:  session.CardsCorrect,
			AccuracyPct:   service.SessionAccuracyPct(session),
			Duration:      service.SessionDuration(session),
		},
	})
}

// GetDeckProgress — GET /api/decks/:id/progress
func (h *StudyHandler) GetDeckProgress(c *gin.Context) {
	deckID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		BadRequestSimple(c, "неверный ID набора")
		return
	}
	userID := middleware.GetUserID(c)

	progress, err := h.studyService.GetDeckProgress(c.Request.Context(), userID, deckID)
	if err != nil {
		if err == service.ErrDeckNotFound {
			NotFound(c, err.Error())
			return
		}
		if err == service.ErrDeckForbidden {
			Forbidden(c, err.Error())
			return
		}
		InternalError(c, "ошибка загрузки прогресса")
		return
	}
	JSON(c, progress)
}

// ListSessions — GET /api/study/sessions
func (h *StudyHandler) ListSessions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	limit := 20
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}

	sessions, err := h.studyService.ListSessions(c.Request.Context(), userID, limit)
	if err != nil {
		InternalError(c, "ошибка загрузки сессий")
		return
	}
	dtos := make([]domain.StudySessionDTO, 0, len(sessions))
	for i := range sessions {
		dtos = append(dtos, sessionToDTO(&sessions[i]))
	}
	JSON(c, gin.H{"sessions": dtos})
}

// sessionToDTO конвертирует domain.StudySession в API-DTO.
func sessionToDTO(s *domain.StudySession) domain.StudySessionDTO {
	dto := domain.StudySessionDTO{
		ID:            s.ID,
		DeckID:        s.DeckID,
		CardsTotal:    s.CardsTotal,
		CardsReviewed: s.CardsReviewed,
		CardsCorrect:  s.CardsCorrect,
		Status:        s.Status,
		StartedAt:     s.StartedAt.Format(time.RFC3339),
	}
	if s.EndedAt != nil {
		dto.EndedAt = s.EndedAt.Format(time.RFC3339)
	}
	return dto
}
