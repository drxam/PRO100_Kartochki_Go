package service

import (
	"context"
	"errors"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
)

var (
	ErrAlreadyFavorite = errors.New("набор уже в избранном")
	ErrNotFavorite     = errors.New("набора нет в избранном")
)

// FavoriteStore — контракт репозитория избранного.
type FavoriteStore interface {
	Add(ctx context.Context, userID, deckID int) error
	Remove(ctx context.Context, userID, deckID int) error
	IsFavorite(ctx context.Context, userID, deckID int) (bool, error)
	ListByUser(ctx context.Context, userID, page, limit int) ([]domain.Deck, int, error)
}

// FavoriteDeckStore — минимальный контракт набора для проверки существования/доступа.
type FavoriteDeckStore interface {
	GetByID(ctx context.Context, id int) (*domain.Deck, error)
}

type FavoriteService struct {
	favoriteRepo FavoriteStore
	deckRepo     FavoriteDeckStore
}

func NewFavoriteService(favoriteRepo FavoriteStore, deckRepo FavoriteDeckStore) *FavoriteService {
	return &FavoriteService{favoriteRepo: favoriteRepo, deckRepo: deckRepo}
}

// Add добавляет набор в избранное.
func (s *FavoriteService) Add(ctx context.Context, userID, deckID int) error {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil || deck == nil {
		return ErrDeckNotFound
	}
	// В избранное можно добавить только публичные наборы (или свои)
	if !deck.IsPublic && deck.UserID != userID {
		return ErrDeckForbidden
	}

	already, err := s.favoriteRepo.IsFavorite(ctx, userID, deckID)
	if err != nil {
		return err
	}
	if already {
		return ErrAlreadyFavorite
	}
	return s.favoriteRepo.Add(ctx, userID, deckID)
}

// Remove убирает набор из избранного.
func (s *FavoriteService) Remove(ctx context.Context, userID, deckID int) error {
	exists, err := s.favoriteRepo.IsFavorite(ctx, userID, deckID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrNotFavorite
	}
	return s.favoriteRepo.Remove(ctx, userID, deckID)
}

// IsFavorite проверяет, добавлен ли набор в избранное.
func (s *FavoriteService) IsFavorite(ctx context.Context, userID, deckID int) (bool, error) {
	return s.favoriteRepo.IsFavorite(ctx, userID, deckID)
}

// List возвращает избранные наборы пользователя с пагинацией.
func (s *FavoriteService) List(ctx context.Context, userID, page, limit int) (*domain.DecksListResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if page <= 0 {
		page = 1
	}

	decks, total, err := s.favoriteRepo.ListByUser(ctx, userID, page, limit)
	if err != nil {
		return nil, err
	}

	items := make([]domain.DeckListItem, 0, len(decks))
	for _, d := range decks {
		items = append(items, domain.DeckListItem{
			ID:          d.ID,
			Title:       d.Title,
			Description: d.Description,
			IsPublic:    d.IsPublic,
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
