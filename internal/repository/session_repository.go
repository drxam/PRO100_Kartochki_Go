package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
)

type SessionRepository struct {
	db *DB
}

func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, s *domain.StudySession) error {
	query := `INSERT INTO study_sessions (user_id, deck_id, cards_total, status)
		VALUES ($1,$2,$3,'active') RETURNING id, started_at, created_at`
	return r.db.Pool.QueryRow(ctx, query, s.UserID, s.DeckID, s.CardsTotal).
		Scan(&s.ID, &s.StartedAt, &s.CreatedAt)
}

func (r *SessionRepository) GetByID(ctx context.Context, id int) (*domain.StudySession, error) {
	query := `SELECT id, user_id, deck_id, started_at, ended_at, cards_total,
		cards_reviewed, cards_correct, status, created_at
		FROM study_sessions WHERE id=$1`
	var s domain.StudySession
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.UserID, &s.DeckID, &s.StartedAt, &s.EndedAt,
		&s.CardsTotal, &s.CardsReviewed, &s.CardsCorrect, &s.Status, &s.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

// IncrementReviewed увеличивает счётчик просмотренных карточек (+correct если правильно).
func (r *SessionRepository) IncrementReviewed(ctx context.Context, sessionID int, correct bool) error {
	if correct {
		_, err := r.db.Pool.Exec(ctx,
			`UPDATE study_sessions SET cards_reviewed=cards_reviewed+1, cards_correct=cards_correct+1 WHERE id=$1`,
			sessionID)
		return err
	}
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE study_sessions SET cards_reviewed=cards_reviewed+1 WHERE id=$1`,
		sessionID)
	return err
}

// Finish завершает сессию.
func (r *SessionRepository) Finish(ctx context.Context, sessionID int) error {
	now := time.Now()
	_, err := r.db.Pool.Exec(ctx,
		`UPDATE study_sessions SET status='completed', ended_at=$2 WHERE id=$1`,
		sessionID, now)
	return err
}

// ListByUser возвращает историю сессий пользователя.
func (r *SessionRepository) ListByUser(ctx context.Context, userID, limit int) ([]domain.StudySession, error) {
	query := `SELECT id, user_id, deck_id, started_at, ended_at, cards_total,
		cards_reviewed, cards_correct, status, created_at
		FROM study_sessions WHERE user_id=$1 ORDER BY started_at DESC LIMIT $2`
	rows, err := r.db.Pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []domain.StudySession
	for rows.Next() {
		var s domain.StudySession
		if err := rows.Scan(
			&s.ID, &s.UserID, &s.DeckID, &s.StartedAt, &s.EndedAt,
			&s.CardsTotal, &s.CardsReviewed, &s.CardsCorrect, &s.Status, &s.CreatedAt,
		); err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}
