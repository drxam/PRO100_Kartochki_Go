package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/sm2"
)

var (
	ErrSessionNotFound  = errors.New("сессия не найдена")
	ErrSessionForbidden = errors.New("нет доступа к сессии")
	ErrSessionCompleted = errors.New("сессия уже завершена")
	ErrNothingToReview  = errors.New("нет карточек для повторения")
)

// StudyProgressStore — контракт репозитория прогресса.
type StudyProgressStore interface {
	Upsert(ctx context.Context, p *domain.CardProgress) error
	GetByUserCard(ctx context.Context, userID, cardID int) (*domain.CardProgress, error)
	ListDueByDeck(ctx context.Context, userID, deckID int) ([]domain.Card, error)
	GetDeckStats(ctx context.Context, userID, deckID int) (*domain.DeckProgressStats, error)
	GetNextDueCard(ctx context.Context, userID, deckID int, reviewedAfter time.Time) (*domain.Card, error)
	GetUserStats(ctx context.Context, userID int) (*domain.UserProgressStats, error)
}

// StudySessionStore — контракт репозитория сессий.
type StudySessionStore interface {
	Create(ctx context.Context, s *domain.StudySession) error
	GetByID(ctx context.Context, id int) (*domain.StudySession, error)
	IncrementReviewed(ctx context.Context, sessionID int, correct bool) error
	Finish(ctx context.Context, sessionID int) error
	ListByUser(ctx context.Context, userID, limit int) ([]domain.StudySession, error)
}

// StudyDeckStore — минимальный контракт для получения набора.
type StudyDeckStore interface {
	GetByID(ctx context.Context, id int) (*domain.Deck, error)
}

// StudyCardStore — минимальный контракт для получения карточки.
type StudyCardStore interface {
	GetByID(ctx context.Context, id int) (*domain.Card, error)
}

type StudyService struct {
	progressRepo StudyProgressStore
	sessionRepo  StudySessionStore
	deckRepo     StudyDeckStore
	cardRepo     StudyCardStore
}

func NewStudyService(
	progressRepo StudyProgressStore,
	sessionRepo StudySessionStore,
	deckRepo StudyDeckStore,
	cardRepo StudyCardStore,
) *StudyService {
	return &StudyService{
		progressRepo: progressRepo,
		sessionRepo:  sessionRepo,
		deckRepo:     deckRepo,
		cardRepo:     cardRepo,
	}
}

// StartSession начинает сессию обучения по набору.
func (s *StudyService) StartSession(ctx context.Context, userID, deckID int) (*domain.StudySession, *domain.Card, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil || deck == nil {
		return nil, nil, ErrDeckNotFound
	}
	if !deck.IsPublic && deck.UserID != userID {
		return nil, nil, ErrDeckForbidden
	}

	dueCards, err := s.progressRepo.ListDueByDeck(ctx, userID, deckID)
	if err != nil {
		return nil, nil, err
	}
	if len(dueCards) == 0 {
		return nil, nil, ErrNothingToReview
	}

	session := &domain.StudySession{
		UserID:     userID,
		DeckID:     deckID,
		CardsTotal: len(dueCards),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, nil, err
	}

	return session, &dueCards[0], nil
}

// ReviewCard применяет SM-2 к карточке и возвращает следующую.
func (s *StudyService) ReviewCard(ctx context.Context, userID, sessionID, cardID, quality int) (*domain.StudySession, *domain.Card, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil || session == nil {
		return nil, nil, ErrSessionNotFound
	}
	if session.UserID != userID {
		return nil, nil, ErrSessionForbidden
	}
	if session.Status == domain.SessionStatusCompleted {
		return nil, nil, ErrSessionCompleted
	}

	card, err := s.cardRepo.GetByID(ctx, cardID)
	if err != nil || card == nil {
		return nil, nil, ErrCardNotFound
	}

	// Получаем текущий прогресс или создаём новый
	prog, err := s.progressRepo.GetByUserCard(ctx, userID, cardID)
	if err != nil {
		return nil, nil, err
	}
	if prog == nil {
		prog = &domain.CardProgress{
			UserID:       userID,
			CardID:       cardID,
			DeckID:       session.DeckID,
			EaseFactor:   sm2.InitEaseFactor,
			IntervalDays: 0,
			Repetitions:  0,
			Status:       domain.ProgressStatusNew,
		}
	}

	// Применяем SM-2
	result := sm2.Review(prog.EaseFactor, prog.IntervalDays, prog.Repetitions, quality)
	now := time.Now()
	prog.EaseFactor = result.EaseFactor
	prog.IntervalDays = result.IntervalDays
	prog.Repetitions = result.Repetitions
	prog.NextReviewAt = result.NextReviewAt
	prog.LastReviewedAt = &now
	prog.Status = result.Status

	if err := s.progressRepo.Upsert(ctx, prog); err != nil {
		return nil, nil, err
	}

	// Обновляем счётчик сессии
	correct := sm2.IsCorrect(quality)
	if err := s.sessionRepo.IncrementReviewed(ctx, sessionID, correct); err != nil {
		return nil, nil, err
	}

	// Получаем обновлённую сессию
	session, err = s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	// Следующая карточка (не трогали после начала сессии)
	next, err := s.progressRepo.GetNextDueCard(ctx, userID, session.DeckID, session.StartedAt)
	if err != nil {
		return nil, nil, err
	}

	// Если карточек больше нет — завершаем сессию
	if next == nil {
		if err := s.sessionRepo.Finish(ctx, sessionID); err != nil {
			return nil, nil, err
		}
		session.Status = domain.SessionStatusCompleted
		endedAt := time.Now()
		session.EndedAt = &endedAt
	}

	return session, next, nil
}

// FinishSession явно завершает сессию (идемпотентно).
func (s *StudyService) FinishSession(ctx context.Context, userID, sessionID int) (*domain.StudySession, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil || session == nil {
		return nil, ErrSessionNotFound
	}
	if session.UserID != userID {
		return nil, ErrSessionForbidden
	}
	if session.Status == domain.SessionStatusCompleted {
		return session, nil
	}
	if err := s.sessionRepo.Finish(ctx, sessionID); err != nil {
		return nil, err
	}
	session.Status = domain.SessionStatusCompleted
	return session, nil
}

// GetDeckProgress возвращает статистику прогресса по набору.
func (s *StudyService) GetDeckProgress(ctx context.Context, userID, deckID int) (*domain.DeckProgressResponse, error) {
	deck, err := s.deckRepo.GetByID(ctx, deckID)
	if err != nil || deck == nil {
		return nil, ErrDeckNotFound
	}
	if !deck.IsPublic && deck.UserID != userID {
		return nil, ErrDeckForbidden
	}

	stats, err := s.progressRepo.GetDeckStats(ctx, userID, deckID)
	if err != nil {
		return nil, err
	}
	return &domain.DeckProgressResponse{
		DeckID:        deckID,
		CardsTotal:    stats.CardsTotal,
		CardsNew:      stats.CardsNew,
		CardsDue:      stats.CardsDue,
		CardsMastered: stats.CardsMastered,
	}, nil
}

// GetSessionByID возвращает сессию (только владельцу).
func (s *StudyService) GetSessionByID(ctx context.Context, userID, sessionID int) (*domain.StudySession, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil || session == nil {
		return nil, ErrSessionNotFound
	}
	if session.UserID != userID {
		return nil, ErrSessionForbidden
	}
	return session, nil
}

// ListSessions возвращает историю сессий пользователя.
func (s *StudyService) ListSessions(ctx context.Context, userID, limit int) ([]domain.StudySession, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	return s.sessionRepo.ListByUser(ctx, userID, limit)
}

// SessionAccuracyPct вычисляет процент правильных ответов в сессии.
func SessionAccuracyPct(session *domain.StudySession) float64 {
	if session.CardsReviewed == 0 {
		return 0
	}
	return float64(session.CardsCorrect) / float64(session.CardsReviewed) * 100
}

// SessionDuration возвращает продолжительность сессии в виде строки.
func SessionDuration(session *domain.StudySession) string {
	end := time.Now()
	if session.EndedAt != nil {
		end = *session.EndedAt
	}
	total := int(end.Sub(session.StartedAt).Seconds())
	if total < 60 {
		return fmt.Sprintf("%dс", total)
	}
	return fmt.Sprintf("%dм %dс", total/60, total%60)
}
