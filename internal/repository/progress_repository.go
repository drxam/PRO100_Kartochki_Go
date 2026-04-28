package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
)

type ProgressRepository struct {
	db *DB
}

func NewProgressRepository(db *DB) *ProgressRepository {
	return &ProgressRepository{db: db}
}

// Upsert создаёт или обновляет запись прогресса для карточки.
func (r *ProgressRepository) Upsert(ctx context.Context, p *domain.CardProgress) error {
	query := `
		INSERT INTO card_progress
			(user_id, card_id, deck_id, ease_factor, interval_days, repetitions,
			 next_review_at, last_reviewed_at, status, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,NOW())
		ON CONFLICT (user_id, card_id) DO UPDATE SET
			ease_factor      = EXCLUDED.ease_factor,
			interval_days    = EXCLUDED.interval_days,
			repetitions      = EXCLUDED.repetitions,
			next_review_at   = EXCLUDED.next_review_at,
			last_reviewed_at = EXCLUDED.last_reviewed_at,
			status           = EXCLUDED.status,
			updated_at       = NOW()
		RETURNING id, created_at, updated_at`
	return r.db.Pool.QueryRow(ctx, query,
		p.UserID, p.CardID, p.DeckID,
		p.EaseFactor, p.IntervalDays, p.Repetitions,
		p.NextReviewAt, p.LastReviewedAt, p.Status,
	).Scan(&p.ID, &p.CreatedAt, &p.UpdatedAt)
}

// GetByUserCard возвращает прогресс пользователя по карточке.
func (r *ProgressRepository) GetByUserCard(ctx context.Context, userID, cardID int) (*domain.CardProgress, error) {
	var p domain.CardProgress
	err := r.db.Pool.QueryRow(ctx,
		`SELECT id, user_id, card_id, deck_id, ease_factor, interval_days, repetitions,
		 next_review_at, last_reviewed_at, status, created_at, updated_at
		 FROM card_progress WHERE user_id=$1 AND card_id=$2`,
		userID, cardID,
	).Scan(
		&p.ID, &p.UserID, &p.CardID, &p.DeckID,
		&p.EaseFactor, &p.IntervalDays, &p.Repetitions,
		&p.NextReviewAt, &p.LastReviewedAt, &p.Status,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// ListDueByDeck возвращает карточки набора, которые нужно повторить прямо сейчас
// (новые + просроченные).
func (r *ProgressRepository) ListDueByDeck(ctx context.Context, userID, deckID int) ([]domain.Card, error) {
	query := `
		SELECT c.id, c.deck_id, c.question, c.answer, c.category_id, c.created_at, c.updated_at
		FROM cards c
		WHERE c.deck_id = $1
		  AND (
		    NOT EXISTS (SELECT 1 FROM card_progress cp WHERE cp.card_id=c.id AND cp.user_id=$2)
		    OR EXISTS (
		        SELECT 1 FROM card_progress cp
		        WHERE cp.card_id=c.id AND cp.user_id=$2 AND cp.next_review_at <= NOW()
		    )
		  )
		ORDER BY
		    CASE WHEN EXISTS (SELECT 1 FROM card_progress cp WHERE cp.card_id=c.id AND cp.user_id=$2)
		         THEN 0 ELSE 1 END, -- просроченные первыми
		    c.id`
	rows, err := r.db.Pool.Query(ctx, query, deckID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanDueCards(rows)
}

// GetDeckStats возвращает статистику прогресса по набору для пользователя.
func (r *ProgressRepository) GetDeckStats(ctx context.Context, userID, deckID int) (*domain.DeckProgressStats, error) {
	var stats domain.DeckProgressStats

	if err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM cards WHERE deck_id=$1`, deckID,
	).Scan(&stats.CardsTotal); err != nil {
		return nil, err
	}

	if err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM cards c WHERE c.deck_id=$1
		 AND NOT EXISTS (SELECT 1 FROM card_progress cp WHERE cp.card_id=c.id AND cp.user_id=$2)`,
		deckID, userID,
	).Scan(&stats.CardsNew); err != nil {
		return nil, err
	}

	if err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM card_progress
		 WHERE deck_id=$1 AND user_id=$2 AND next_review_at<=NOW() AND status!='mastered'`,
		deckID, userID,
	).Scan(&stats.CardsDue); err != nil {
		return nil, err
	}

	if err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM card_progress WHERE deck_id=$1 AND user_id=$2 AND status='mastered'`,
		deckID, userID,
	).Scan(&stats.CardsMastered); err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetNextDueCard возвращает следующую карточку для повторения
// (которую не трогали после reviewedAfter).
func (r *ProgressRepository) GetNextDueCard(ctx context.Context, userID, deckID int, reviewedAfter time.Time) (*domain.Card, error) {
	query := `
		SELECT c.id, c.deck_id, c.question, c.answer, c.category_id, c.created_at, c.updated_at
		FROM cards c
		WHERE c.deck_id = $1
		  AND (
		    NOT EXISTS (SELECT 1 FROM card_progress cp WHERE cp.card_id=c.id AND cp.user_id=$2)
		    OR EXISTS (
		        SELECT 1 FROM card_progress cp
		        WHERE cp.card_id=c.id AND cp.user_id=$2
		          AND cp.next_review_at <= NOW()
		          AND (cp.last_reviewed_at IS NULL OR cp.last_reviewed_at < $3)
		    )
		  )
		ORDER BY c.id
		LIMIT 1`
	var c domain.Card
	err := r.db.Pool.QueryRow(ctx, query, deckID, userID, reviewedAfter).Scan(
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

// GetUserStats возвращает общую статистику пользователя.
func (r *ProgressRepository) GetUserStats(ctx context.Context, userID int) (*domain.UserProgressStats, error) {
	var stats domain.UserProgressStats
	err := r.db.Pool.QueryRow(ctx,
		`SELECT COUNT(*),
		 COALESCE(SUM(CASE WHEN status='mastered' THEN 1 ELSE 0 END),0),
		 COALESCE(SUM(CASE WHEN next_review_at<=NOW() AND status!='mastered' THEN 1 ELSE 0 END),0)
		 FROM card_progress WHERE user_id=$1`, userID,
	).Scan(&stats.TotalReviewed, &stats.TotalMastered, &stats.TotalDue)
	return &stats, err
}

// scanDueCards — хелпер для сканирования строк карточек из запросов прогресса.
func scanDueCards(rows pgx.Rows) ([]domain.Card, error) {
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
