package repository

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
)

type CardRepository struct {
	db *DB
}

func NewCardRepository(db *DB) *CardRepository {
	return &CardRepository{db: db}
}

func (r *CardRepository) Create(ctx context.Context, c *domain.Card) error {
	query := `INSERT INTO cards (deck_id, question, answer, category_id)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at`
	return r.db.Pool.QueryRow(ctx, query, c.DeckID, c.Question, c.Answer, c.CategoryID).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

func (r *CardRepository) GetByID(ctx context.Context, id int) (*domain.Card, error) {
	query := `SELECT id, deck_id, question, answer, category_id, created_at, updated_at FROM cards WHERE id = $1`
	var c domain.Card
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.DeckID, &c.Question, &c.Answer, &c.CategoryID, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *CardRepository) ListByDeckID(ctx context.Context, deckID int) ([]domain.Card, error) {
	query := `SELECT id, deck_id, question, answer, category_id, created_at, updated_at
		FROM cards WHERE deck_id = $1 ORDER BY created_at`
	rows, err := r.db.Pool.Query(ctx, query, deckID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanCards(rows)
}

func (r *CardRepository) CountByDeckID(ctx context.Context, deckID int) (int, error) {
	var n int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM cards WHERE deck_id = $1`, deckID).Scan(&n)
	return n, err
}

// CountByUserID возвращает общее количество карточек во всех наборах пользователя.
func (r *CardRepository) CountByUserID(ctx context.Context, userID int) (int, error) {
	var n int
	err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*) FROM cards c INNER JOIN decks d ON c.deck_id = d.id WHERE d.user_id = $1`, userID).Scan(&n)
	return n, err
}

// ListByUserIDWithFilters возвращает карточки пользователя с пагинацией и фильтрами.
func (r *CardRepository) ListByUserIDWithFilters(ctx context.Context, userID int, page, limit int, categoryID *int, tagID *int, search string) ([]domain.Card, int, error) {
	baseCond := ` FROM cards c INNER JOIN decks d ON c.deck_id = d.id WHERE d.user_id = $1`
	args := []interface{}{userID}
	pos := 2
	if categoryID != nil {
		baseCond += ` AND c.category_id = $` + strconv.Itoa(pos)
		args = append(args, *categoryID)
		pos++
	}
	if tagID != nil {
		baseCond += ` AND EXISTS (SELECT 1 FROM card_tags ct WHERE ct.card_id = c.id AND ct.tag_id = $` + strconv.Itoa(pos) + `)`
		args = append(args, *tagID)
		pos++
	}
	if search != "" {
		baseCond += ` AND (c.question ILIKE $` + strconv.Itoa(pos) + ` OR c.answer ILIKE $` + strconv.Itoa(pos) + `)`
		args = append(args, "%"+search+"%")
		pos++
	}
	var total int
	if err := r.db.Pool.QueryRow(ctx, `SELECT COUNT(*)`+baseCond, args...).Scan(&total); err != nil {
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
	offset := (page - 1) * limit
	listArgs := append([]interface{}{}, args...)
	listArgs = append(listArgs, limit, offset)
	listQuery := `SELECT c.id, c.deck_id, c.question, c.answer, c.category_id, c.created_at, c.updated_at` + baseCond +
		` ORDER BY c.created_at DESC LIMIT $` + strconv.Itoa(pos) + ` OFFSET $` + strconv.Itoa(pos+1)
	rows, err := r.db.Pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list, err := r.scanCards(rows)
	return list, total, err
}

func (r *CardRepository) Update(ctx context.Context, c *domain.Card) error {
	query := `UPDATE cards SET question=$2, answer=$3, category_id=$4, updated_at=NOW() WHERE id=$1 RETURNING updated_at`
	return r.db.Pool.QueryRow(ctx, query, c.ID, c.Question, c.Answer, c.CategoryID).Scan(&c.UpdatedAt)
}

func (r *CardRepository) Delete(ctx context.Context, id int) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM cards WHERE id = $1`, id)
	return err
}

func (r *CardRepository) SetCardTags(ctx context.Context, cardID int, tagIDs []int) error {
	_, err := r.db.Pool.Exec(ctx, `DELETE FROM card_tags WHERE card_id = $1`, cardID)
	if err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		_, err = r.db.Pool.Exec(ctx, `INSERT INTO card_tags (card_id, tag_id) VALUES ($1, $2)`, cardID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *CardRepository) GetCardTagIDs(ctx context.Context, cardID int) ([]int, error) {
	rows, err := r.db.Pool.Query(ctx, `SELECT tag_id FROM card_tags WHERE card_id = $1`, cardID)
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

func (r *CardRepository) scanCards(rows pgx.Rows) ([]domain.Card, error) {
	var list []domain.Card
	for rows.Next() {
		var c domain.Card
		if err := rows.Scan(&c.ID, &c.DeckID, &c.Question, &c.Answer, &c.CategoryID, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, rows.Err()
}
