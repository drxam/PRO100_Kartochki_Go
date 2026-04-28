package domain

import "time"

// CardProgress — прогресс пользователя по конкретной карточке (SM-2).
type CardProgress struct {
	ID             int        `json:"id"`
	UserID         int        `json:"user_id"`
	CardID         int        `json:"card_id"`
	DeckID         int        `json:"deck_id"`
	EaseFactor     float64    `json:"ease_factor"`
	IntervalDays   int        `json:"interval_days"`
	Repetitions    int        `json:"repetitions"`
	NextReviewAt   time.Time  `json:"next_review_at"`
	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty"`
	Status         string     `json:"status"` // new | learning | review | mastered
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

const (
	ProgressStatusNew      = "new"
	ProgressStatusLearning = "learning"
	ProgressStatusReview   = "review"
	ProgressStatusMastered = "mastered"
)

// StudySession — сессия обучения по набору карточек.
type StudySession struct {
	ID            int        `json:"id"`
	UserID        int        `json:"user_id"`
	DeckID        int        `json:"deck_id"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	CardsTotal    int        `json:"cards_total"`
	CardsReviewed int        `json:"cards_reviewed"`
	CardsCorrect  int        `json:"cards_correct"`
	Status        string     `json:"status"` // active | completed
	CreatedAt     time.Time  `json:"created_at"`
}

const (
	SessionStatusActive    = "active"
	SessionStatusCompleted = "completed"
)

// DeckProgressStats — статистика прогресса пользователя по набору.
type DeckProgressStats struct {
	CardsTotal    int
	CardsNew      int
	CardsDue      int
	CardsMastered int
}

// UserProgressStats — общая статистика прогресса пользователя.
type UserProgressStats struct {
	TotalReviewed int
	TotalMastered int
	TotalDue      int
}
