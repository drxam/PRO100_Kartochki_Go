package service

import (
	"context"
	"errors"
	"time"

	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/repository"
)

var (
	ErrCardNotFound   = errors.New("карточка не найдена")
	ErrCardForbidden  = errors.New("нет доступа к карточке")
)

type CardService struct {
	cardRepo     *repository.CardRepository
	deckRepo     *repository.DeckRepository
	categoryRepo *repository.CategoryRepository
	tagRepo      *repository.TagRepository
}

func NewCardService(cardRepo *repository.CardRepository, deckRepo *repository.DeckRepository, categoryRepo *repository.CategoryRepository, tagRepo *repository.TagRepository) *CardService {
	return &CardService{
		cardRepo:     cardRepo,
		deckRepo:     deckRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
	}
}

func (s *CardService) Create(ctx context.Context, deckID int, userID int, req domain.CreateCardRequest) (*domain.Card, error) {
	if deckID == 0 && req.DeckID != nil {
		deckID = *req.DeckID
	}
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil || deck == nil || deck.UserID != userID {
		return nil, ErrCardForbidden
	}
	c := &domain.Card{
		DeckID:     deckID,
		Question:   req.Question,
		Answer:     req.Answer,
		CategoryID: req.CategoryID,
	}
	if err := s.cardRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	if len(req.TagIDs) > 0 {
		_ = s.cardRepo.SetCardTags(ctx, c.ID, req.TagIDs)
		c.Tags, _ = s.tagRepo.GetByIDs(ctx, req.TagIDs)
	}
	if c.CategoryID != nil {
		c.Category, _ = s.categoryRepo.GetByID(ctx, *c.CategoryID)
	}
	return c, nil
}

func (s *CardService) GetByID(ctx context.Context, id int, userID int) (*domain.Card, error) {
	c, err := s.cardRepo.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCardNotFound
	}
	deck, _ := s.deckRepo.GetByID(ctx, c.DeckID)
	if deck == nil || (!deck.IsPublic && deck.UserID != userID) {
		return nil, ErrCardForbidden
	}
	tagIDs, _ := s.cardRepo.GetCardTagIDs(ctx, c.ID)
	if len(tagIDs) > 0 {
		c.Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
	}
	if c.CategoryID != nil {
		c.Category, _ = s.categoryRepo.GetByID(ctx, *c.CategoryID)
	}
	return c, nil
}

func (s *CardService) ListByDeck(ctx context.Context, deckID int, userID int) ([]domain.Card, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil || deck == nil || (!deck.IsPublic && deck.UserID != userID) {
		return nil, ErrCardForbidden
	}
	list, err := s.cardRepo.ListByDeckID(ctx, deckID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		tagIDs, _ := s.cardRepo.GetCardTagIDs(ctx, list[i].ID)
		if len(tagIDs) > 0 {
			list[i].Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
		}
		if list[i].CategoryID != nil {
			list[i].Category, _ = s.categoryRepo.GetByID(ctx, *list[i].CategoryID)
		}
	}
	return list, nil
}

func (s *CardService) Update(ctx context.Context, id int, userID int, req domain.UpdateCardRequest) (*domain.Card, error) {
	c, err := s.cardRepo.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCardNotFound
	}
	deck, _ := s.deckRepo.GetByID(ctx, c.DeckID)
	if deck == nil || deck.UserID != userID {
		return nil, ErrCardForbidden
	}
	if req.Question != nil {
		c.Question = *req.Question
	}
	if req.Answer != nil {
		c.Answer = *req.Answer
	}
	if req.CategoryID != nil {
		c.CategoryID = req.CategoryID
	}
	if err := s.cardRepo.Update(ctx, c); err != nil {
		return nil, err
	}
	if req.TagIDs != nil {
		_ = s.cardRepo.SetCardTags(ctx, c.ID, req.TagIDs)
		c.Tags, _ = s.tagRepo.GetByIDs(ctx, req.TagIDs)
	} else {
		tagIDs, _ := s.cardRepo.GetCardTagIDs(ctx, c.ID)
		if len(tagIDs) > 0 {
			c.Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
		}
	}
	if c.CategoryID != nil {
		c.Category, _ = s.categoryRepo.GetByID(ctx, *c.CategoryID)
	}
	return c, nil
}

func (s *CardService) Delete(ctx context.Context, id int, userID int) error {
	c, err := s.cardRepo.GetByID(ctx, id)
	if err != nil || c == nil {
		return ErrCardNotFound
	}
	deck, _ := s.deckRepo.GetByID(ctx, c.DeckID)
	if deck == nil || deck.UserID != userID {
		return ErrCardForbidden
	}
	return s.cardRepo.Delete(ctx, id)
}

// ListByUserPaginated возвращает карточки пользователя с пагинацией и фильтрами.
func (s *CardService) ListByUserPaginated(ctx context.Context, userID int, page, limit int, categoryID *int, tagID *int, search string) (*domain.CardsListResponse, error) {
	if limit > 100 {
		limit = 100
	}
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	list, total, err := s.cardRepo.ListByUserIDWithFilters(ctx, userID, page, limit, categoryID, tagID, search)
	if err != nil {
		return nil, err
	}
	items := make([]domain.CardListItem, 0, len(list))
	for _, c := range list {
		deck, _ := s.deckRepo.GetByID(ctx, c.DeckID)
		item := domain.CardListItem{
			ID:        c.ID,
			Question:  c.Question,
			Answer:    c.Answer,
			CreatedAt: c.CreatedAt.Format(time.RFC3339),
		}
		if deck != nil {
			item.Deck = domain.DeckBrief{ID: deck.ID, Title: deck.Title}
		}
		if c.CategoryID != nil {
			item.Category, _ = s.categoryRepo.GetByID(ctx, *c.CategoryID)
		}
		tagIDs, _ := s.cardRepo.GetCardTagIDs(ctx, c.ID)
		if len(tagIDs) > 0 {
			item.Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
		}
		items = append(items, item)
	}
	return &domain.CardsListResponse{
		Cards: items,
		Pagination: domain.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	}, nil
}

// GetByIDForAPI возвращает карточку в формате API (с deck, category, tags).
func (s *CardService) GetByIDForAPI(ctx context.Context, id int, userID int) (*domain.CardListItem, error) {
	c, err := s.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	deck, _ := s.deckRepo.GetByID(ctx, c.DeckID)
	item := domain.CardListItem{
		ID:        c.ID,
		Question:  c.Question,
		Answer:    c.Answer,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
	}
	if deck != nil {
		item.Deck = domain.DeckBrief{ID: deck.ID, Title: deck.Title}
	}
	item.Category = c.Category
	item.Tags = c.Tags
	return &item, nil
}
