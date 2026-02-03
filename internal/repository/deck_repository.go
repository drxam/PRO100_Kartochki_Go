package repository

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
)

type DeckRepository struct {
	db *DB
}

func NewDeckRepository(db *DB) *DeckRepository {
	return &DeckRepository{db: db}
}

func (r *DeckRepository) Create(ctx context.Context, d *domain.Deck) error {
	query := `INSERT INTO decks (user_id, title, description, category_id, is_public)
		VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at, updated_at`
	return r.db.Pool.QueryRow(ctx, query,
		d.UserID, d.Title, d.Description, d.CategoryID, d.IsPublic,
	).Scan(&d.ID, &d.CreatedAt, &d.UpdatedAt)
}

func (r *DeckRepository) GetByID(ctx context.Context, id int) (*domain.Deck, error) {
	query := `SELECT id, user_id, title, description, category_id, is_public, created_at, updated_at
		FROM decks WHERE id = $1`
	var d domain.Deck
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&d.ID, &d.UserID, &d.Title, &d.Description, &d.CategoryID, &d.IsPublic,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *DeckRepository) ListByUserID(ctx context.Context, userID int) ([]domain.Deck, error) {
	query := `SELECT id, user_id, title, description, category_id, is_public, created_at, updated_at
		FROM decks WHERE user_id = $1 ORDER BY updated_at DESC`
	rows, err := r.db.Pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanDecks(rows)
}

// ListByUserIDWithFilters возвращает наборы с пагинацией и опционально category_id, search.
func (r *DeckRepository) ListByUserIDWithFilters(ctx context.Context, userID int, page, limit int, categoryID *int, search string) ([]domain.Deck, int, error) {
	baseCond := ` WHERE user_id = $1`
	args := []interface{}{userID}
	pos := 2
	if categoryID != nil {
		baseCond += ` AND category_id = $` + strconv.Itoa(pos)
		args = append(args, *categoryID)
		pos++
	}
	if search != "" {
		baseCond += ` AND (title ILIKE $` + strconv.Itoa(pos) + ` OR description ILIKE $` + strconv.Itoa(pos) + `)`
		args = append(args, "%"+search+"%")
		pos++
	}
	var total int
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM decks`+baseCond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit
	listArgs := append([]interface{}{}, args...)
	listArgs = append(listArgs, limit, offset)
	listQuery := `SELECT id, user_id, title, description, category_id, is_public, created_at, updated_at
		FROM decks` + baseCond + ` ORDER BY updated_at DESC LIMIT $` + strconv.Itoa(pos) + ` OFFSET $` + strconv.Itoa(pos+1)
	rows, err := r.db.Pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list, err := r.scanDecks(rows)
	return list, total, err
}

func (r *DeckRepository) ListPublic(ctx context.Context, limit, offset int) ([]domain.Deck, error) {
	query := `SELECT id, user_id, title, description, category_id, is_public, created_at, updated_at
		FROM decks WHERE is_public = true ORDER BY updated_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanDecks(rows)
}

// ListPublicWithFilters — публичные наборы с пагинацией, фильтрами и сортировкой.
// sortBy: recent (updated_at DESC), popular (cards_count DESC), cards_count (cards_count DESC).
func (r *DeckRepository) ListPublicWithFilters(ctx context.Context, page, limit int, categoryID *int, search string, sortBy string) ([]domain.Deck, int, error) {
	baseCond := ` WHERE d.is_public = true`
	args := []interface{}{}
	pos := 1
	if categoryID != nil {
		baseCond += ` AND d.category_id = $` + strconv.Itoa(pos)
		args = append(args, *categoryID)
		pos++
	}
	if search != "" {
		baseCond += ` AND (d.title ILIKE $` + strconv.Itoa(pos) + ` OR d.description ILIKE $` + strconv.Itoa(pos) + `)`
		args = append(args, "%"+search+"%")
		pos++
	}
	fromClause := ` FROM decks d LEFT JOIN (SELECT deck_id, COUNT(*) AS cnt FROM cards GROUP BY deck_id) c ON d.id = c.deck_id` + baseCond
	var total int
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM decks d`+baseCond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if page <= 0 {
		page = 1
	}
	orderBy := ` ORDER BY d.updated_at DESC`
	switch sortBy {
	case "popular", "cards_count":
		orderBy = ` ORDER BY COALESCE(c.cnt, 0) DESC, d.updated_at DESC`
	case "recent":
		orderBy = ` ORDER BY d.updated_at DESC`
	}
	offset := (page - 1) * limit
	listArgs := append([]interface{}{}, args...)
	listArgs = append(listArgs, limit, offset)
	// возвращаем deck + cards_count из join
	listQuery := `SELECT d.id, d.user_id, d.title, d.description, d.category_id, d.is_public, d.created_at, d.updated_at, COALESCE(c.cnt, 0)::int
		` + fromClause + orderBy + ` LIMIT $` + strconv.Itoa(pos) + ` OFFSET $` + strconv.Itoa(pos+1)
	rows, err := r.db.Pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []domain.Deck
	for rows.Next() {
		var d domain.Deck
		var cnt int
		if err := rows.Scan(&d.ID, &d.UserID, &d.Title, &d.Description, &d.CategoryID, &d.IsPublic, &d.CreatedAt, &d.UpdatedAt, &cnt); err != nil {
			return nil, 0, err
		}
		d.CardsCount = cnt
		list = append(list, d)
	}
	return list, total, rows.Err()
}

func (r *DeckRepository) Update(ctx context.Context, d *domain.Deck) error {
	query := `UPDATE decks SET title=$2, description=$3, category_id=$4, is_public=$5, updated_at=NOW()
		WHERE id=$1 RETURNING updated_at`
	return r.db.Pool.QueryRow(ctx, query, d.ID, d.Title, d.Description, d.CategoryID, d.IsPublic).Scan(&d.UpdatedAt)
}

func (r *DeckRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM decks WHERE id = $1`, id)
	return err
}

func (r *DeckRepository) SetDeckTags(ctx context.Context, deckID int, tagIDs []int) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM deck_tags WHERE deck_id = $1`, deckID)
	if err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		_, err = r.db.Pool.Exec(ctx, `INSERT INTO deck_tags (deck_id, tag_id) VALUES ($1, $2)`, deckID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *DeckRepository) CountByUserID(ctx context.Context, userID int) (int, error) {
	var n int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM decks WHERE user_id = $1`, userID).Scan(&n)
	return n, err
}

func (r *DeckRepository) GetDeckTagIDs(ctx context.Context, deckID int) ([]int, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT tag_id FROM deck_tags WHERE deck_id = $1`, deckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *DeckRepository) scanDecks(rows pgx.Rows) ([]domain.Deck, error) {
	var list []domain.Deck
	for rows.Next() {
		var d domain.Deck
		if err := rows.Scan(&d.ID, &d.UserID, &d.Title, &d.Description, &d.CategoryID, &d.IsPublic, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, rows.Err()
}
