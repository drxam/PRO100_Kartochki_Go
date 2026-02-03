package service

import (
	"context"
	"errors"
	"time"

	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/repository"
)

var (
	ErrDeckNotFound  = errors.New("набор не найден")
	ErrDeckForbidden = errors.New("нет доступа к набору")
)

type DeckService struct {
	deckRepo     *repository.DeckRepository
	cardRepo     *repository.CardRepository
	userRepo     *repository.UserRepository
	categoryRepo *repository.CategoryRepository
	tagRepo      *repository.TagRepository
}

func NewDeckService(deckRepo *repository.DeckRepository, cardRepo *repository.CardRepository, userRepo *repository.UserRepository, categoryRepo *repository.CategoryRepository, tagRepo *repository.TagRepository) *DeckService {
	return &DeckService{
		deckRepo:     deckRepo,
		cardRepo:     cardRepo,
		userRepo:     userRepo,
		categoryRepo: categoryRepo,
		tagRepo:      tagRepo,
	}
}

func (s *DeckService) Create(ctx context.Context, userID int, req domain.CreateDeckRequest) (*domain.Deck, error) {
	d := &domain.Deck{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		CategoryID:  req.CategoryID,
		IsPublic:    req.IsPublic,
		CardsCount:  0,
	}
	if err := s.deckRepo.Create(ctx, d); err != nil {
		return nil, err
	}
	if len(req.TagIDs) > 0 {
		_ = s.deckRepo.SetDeckTags(ctx, d.ID, req.TagIDs)
		d.Tags, _ = s.tagRepo.GetByIDs(ctx, req.TagIDs)
	}
	if d.CategoryID != nil {
		d.Category, _ = s.categoryRepo.GetByID(ctx, *d.CategoryID)
	}
	return d, nil
}

func (s *DeckService) GetByID(ctx context.Context, id int, userID int) (*domain.Deck, error) {
	d, err := s.deckRepo.GetByID(ctx, id)
	if err != nil || d == nil {
		return nil, ErrDeckNotFound
	}
	if !d.IsPublic && d.UserID != userID {
		return nil, ErrDeckForbidden
	}
	tagIDs, _ := s.deckRepo.GetDeckTagIDs(ctx, d.ID)
	if len(tagIDs) > 0 {
		d.Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
	}
	if d.CategoryID != nil {
		d.Category, _ = s.categoryRepo.GetByID(ctx, *d.CategoryID)
	}
	d.CardsCount, _ = s.cardRepo.CountByDeckID(ctx, d.ID)
	cards, _ := s.cardRepo.ListByDeckID(ctx, d.ID)
	for i := range cards {
		if cards[i].CategoryID != nil {
			cards[i].Category, _ = s.categoryRepo.GetByID(ctx, *cards[i].CategoryID)
		}
		tagIDs, _ := s.cardRepo.GetCardTagIDs(ctx, cards[i].ID)
		if len(tagIDs) > 0 {
			cards[i].Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
		}
	}
	d.Cards = cards
	return d, nil
}

func (s *DeckService) ListByUser(ctx context.Context, userID int) ([]domain.Deck, error) {
	list, err := s.deckRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range list {
		tagIDs, _ := s.deckRepo.GetDeckTagIDs(ctx, list[i].ID)
		if len(tagIDs) > 0 {
			list[i].Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
		}
		if list[i].CategoryID != nil {
			list[i].Category, _ = s.categoryRepo.GetByID(ctx, *list[i].CategoryID)
		}
		list[i].CardsCount, _ = s.cardRepo.CountByDeckID(ctx, list[i].ID)
	}
	return list, nil
}

// ListByUserPaginated возвращает наборы с пагинацией и фильтрами.
func (s *DeckService) ListByUserPaginated(ctx context.Context, userID int, page, limit int, categoryID *int, search string) (*domain.DecksListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	list, total, err := s.deckRepo.ListByUserIDWithFilters(ctx, userID, page, limit, categoryID, search)
	if err != nil {
		return nil, err
	}
	items := make([]domain.DeckListItem, 0, len(list))
	for _, d := range list {
		tagIDs, _ := s.deckRepo.GetDeckTagIDs(ctx, d.ID)
		if len(tagIDs) > 0 {
			d.Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
		}
		if d.CategoryID != nil {
			d.Category, _ = s.categoryRepo.GetByID(ctx, *d.CategoryID)
		}
		cnt, _ := s.cardRepo.CountByDeckID(ctx, d.ID)
		items = append(items, domain.DeckListItem{
			ID:          d.ID,
			Title:       d.Title,
			Description: d.Description,
			Category:    d.Category,
			Tags:        d.Tags,
			IsPublic:    d.IsPublic,
			CardsCount:  cnt,
			CreatedAt:   d.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return &domain.DecksListResponse{
		Decks: items,
		Pagination: domain.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	}, nil
}

func (s *DeckService) ListPublic(ctx context.Context, limit, offset int) ([]domain.Deck, error) {
	if limit <= 0 {
		limit = 20
	}
	list, err := s.deckRepo.ListPublic(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	for i := range list {
		tagIDs, _ := s.deckRepo.GetDeckTagIDs(ctx, list[i].ID)
		if len(tagIDs) > 0 {
			list[i].Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
		}
		if list[i].CategoryID != nil {
			list[i].Category, _ = s.categoryRepo.GetByID(ctx, *list[i].CategoryID)
		}
		list[i].CardsCount, _ = s.cardRepo.CountByDeckID(ctx, list[i].ID)
	}
	return list, nil
}

func (s *DeckService) Update(ctx context.Context, id int, userID int, req domain.UpdateDeckRequest) (*domain.Deck, error) {
	d, err := s.deckRepo.GetByID(ctx, id)
	if err != nil || d == nil {
		return nil, ErrDeckNotFound
	}
	if d.UserID != userID {
		return nil, ErrDeckForbidden
	}
	if req.Title != nil {
		d.Title = *req.Title
	}
	if req.Description != nil {
		d.Description = req.Description
	}
	if req.CategoryID != nil {
		d.CategoryID = req.CategoryID
	}
	if req.IsPublic != nil {
		d.IsPublic = *req.IsPublic
	}
	if err := s.deckRepo.Update(ctx, d); err != nil {
		return nil, err
	}
	if req.TagIDs != nil {
		_ = s.deckRepo.SetDeckTags(ctx, d.ID, req.TagIDs)
		d.Tags, _ = s.tagRepo.GetByIDs(ctx, req.TagIDs)
	} else {
		tagIDs, _ := s.deckRepo.GetDeckTagIDs(ctx, d.ID)
		if len(tagIDs) > 0 {
			d.Tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
		}
	}
	if d.CategoryID != nil {
		d.Category, _ = s.categoryRepo.GetByID(ctx, *d.CategoryID)
	}
	d.CardsCount, _ = s.cardRepo.CountByDeckID(ctx, d.ID)
	return d, nil
}

func (s *DeckService) Delete(ctx context.Context, id int, userID int) error {
	d, err := s.deckRepo.GetByID(ctx, id)
	if err != nil || d == nil {
		return ErrDeckNotFound
	}
	if d.UserID != userID {
		return ErrDeckForbidden
	}
	return s.deckRepo.Delete(ctx, id)
}

// ListPublicPaginated — публичные наборы с пагинацией, фильтрами, сортировкой и автором.
func (s *DeckService) ListPublicPaginated(ctx context.Context, page, limit int, categoryID *int, search string, sortBy string) (*domain.PublicDecksListResponse, error) {
	if limit > 100 {
		limit = 100
	}
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	list, total, err := s.deckRepo.ListPublicWithFilters(ctx, page, limit, categoryID, search, sortBy)
	if err != nil {
		return nil, err
	}
	items := make([]domain.PublicDeckListItem, 0, len(list))
	for _, d := range list {
		author := domain.DeckAuthor{}
		if u, _ := s.userRepo.GetByID(ctx, d.UserID); u != nil {
			author.ID = u.ID
			author.Username = u.Username
			author.AvatarURL = u.AvatarURL
		}
		tagIDs, _ := s.deckRepo.GetDeckTagIDs(ctx, d.ID)
		var tags []domain.Tag
		if len(tagIDs) > 0 {
			tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
		}
		var cat *domain.Category
		if d.CategoryID != nil {
			cat, _ = s.categoryRepo.GetByID(ctx, *d.CategoryID)
		}
		items = append(items, domain.PublicDeckListItem{
			ID:          d.ID,
			Title:       d.Title,
			Description: d.Description,
			Category:    cat,
			Tags:        tags,
			CardsCount:  d.CardsCount,
			Author:      author,
			CreatedAt:   d.CreatedAt.Format(time.RFC3339),
		})
	}
	return &domain.PublicDecksListResponse{
		Decks: items,
		Pagination: domain.Pagination{
			Page:  page,
			Limit: limit,
			Total: total,
		},
	}, nil
}

// GetPublicByID возвращает публичный набор по ID (с автором и карточками).
func (s *DeckService) GetPublicByID(ctx context.Context, id int) (*domain.PublicDeckDetail, error) {
	d, err := s.deckRepo.GetByID(ctx, id)
	if err != nil || d == nil || !d.IsPublic {
		return nil, ErrDeckNotFound
	}
	author := domain.DeckAuthor{}
	if u, _ := s.userRepo.GetByID(ctx, d.UserID); u != nil {
		author.ID = u.ID
		author.Username = u.Username
		author.AvatarURL = u.AvatarURL
	}
	tagIDs, _ := s.deckRepo.GetDeckTagIDs(ctx, d.ID)
	var tags []domain.Tag
	if len(tagIDs) > 0 {
		tags, _ = s.tagRepo.GetByIDs(ctx, tagIDs)
	}
	var cat *domain.Category
	if d.CategoryID != nil {
		cat, _ = s.categoryRepo.GetByID(ctx, *d.CategoryID)
	}
	cardsCount, _ := s.cardRepo.CountByDeckID(ctx, d.ID)
	cards, _ := s.cardRepo.ListByDeckID(ctx, d.ID)
	publicCards := make([]domain.PublicCardItem, 0, len(cards))
	for _, c := range cards {
		publicCards = append(publicCards, domain.PublicCardItem{ID: c.ID, Question: c.Question, Answer: c.Answer})
	}
	return &domain.PublicDeckDetail{
		ID:          d.ID,
		Title:       d.Title,
		Description: d.Description,
		Category:    cat,
		Tags:        tags,
		CardsCount:  cardsCount,
		Author:      author,
		Cards:       publicCards,
	}, nil
}
