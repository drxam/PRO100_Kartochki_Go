package repository

import (
	"context"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
)

type FavoriteRepository struct {
	db *DB
}

func NewFavoriteRepository(db *DB) *FavoriteRepository {
	return &FavoriteRepository{db: db}
}

func (r *FavoriteRepository) Add(ctx context.Context, userID, deckID int) error {
	_, err := r.db.Pool.Exec(ctx,
		`INSERT INTO deck_favorites (user_id, deck_id) VALUES ($1,$2) ON CONFLICT DO NOTHING`,
		userID, deckID)
	return err
}

func (r *FavoriteRepository) Remove(ctx context.Context, userID, deckID int) error {
	_, err := r.db.Pool.Exec(ctx,
		`DELETE FROM deck_favorites WHERE user_id=$1 AND deck_id=$2`,
		userID, deckID)
	return err
}

func (r *FavoriteRepository) IsFavorite(ctx context.Context, userID, deckID int) (bool, error) {
	var exists bool
	err := r.db.Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM deck_favorites WHERE user_id=$1 AND deck_id=$2)`,
		userID, deckID).Scan(&exists)
	return exists, err
}

// ListByUser возвращает избранные наборы пользователя с пагинацией.
func (r *FavoriteRepository) ListByUser(ctx context.Context, userID, page, limit int) ([]domain.Deck, int, error) {
	var total int
	if err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM deck_favorites WHERE user_id=$1`, userID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	rows, err := r.db.Pool.Query(ctx,
		`SELECT d.id, d.user_id, d.title, d.description, d.category_id, d.is_public, d.created_at, d.updated_at
		 FROM decks d
		 INNER JOIN deck_favorites f ON d.id = f.deck_id
		 WHERE f.user_id = $1
		 ORDER BY f.created_at DESC
		 LIMIT $2 OFFSET $3`,
		userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []domain.Deck
	for rows.Next() {
		var d domain.Deck
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.Title, &d.Description,
			&d.CategoryID, &d.IsPublic, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, d)
	}
	return list, total, rows.Err()
}
