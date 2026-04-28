package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
)

// ---------------------------------------------------------------------------
// cardStoreMock
// ---------------------------------------------------------------------------

type cardStoreMock struct {
	mu       sync.Mutex
	cards    map[int]*domain.Card
	cardTags map[int][]int // cardID -> []tagID
	nextID   int
}

func newCardStoreMock() *cardStoreMock {
	return &cardStoreMock{
		cards:    make(map[int]*domain.Card),
		cardTags: make(map[int][]int),
		nextID:   1,
	}
}

func (m *cardStoreMock) Create(ctx context.Context, c *domain.Card) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.ID = m.nextID
	m.nextID++
	c.CreatedAt = time.Now()
	c.UpdatedAt = c.CreatedAt
	cp := *c
	m.cards[c.ID] = &cp
	return nil
}

func (m *cardStoreMock) GetByID(ctx context.Context, id int) (*domain.Card, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.cards[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (m *cardStoreMock) ListByDeckID(ctx context.Context, deckID int) ([]domain.Card, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Card
	for _, c := range m.cards {
		if c.DeckID == deckID {
			list = append(list, *c)
		}
	}
	return list, nil
}

func (m *cardStoreMock) CountByDeckID(ctx context.Context, deckID int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, c := range m.cards {
		if c.DeckID == deckID {
			n++
		}
	}
	return n, nil
}

func (m *cardStoreMock) ListByUserIDWithFilters(ctx context.Context, userID int, page, limit int, categoryID *int, tagID *int, search string) ([]domain.Card, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// simplified: returns all cards (no deck ownership filter in mock)
	var list []domain.Card
	for _, c := range m.cards {
		list = append(list, *c)
	}
	total := len(list)
	offset := (page - 1) * limit
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return list[offset:end], total, nil
}

func (m *cardStoreMock) Update(ctx context.Context, c *domain.Card) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.cards[c.ID]
	if !ok {
		return errors.New("not found")
	}
	c.UpdatedAt = time.Now()
	cp := *c
	m.cards[c.ID] = &cp
	return nil
}

func (m *cardStoreMock) Delete(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cards, id)
	delete(m.cardTags, id)
	return nil
}

func (m *cardStoreMock) SetCardTags(ctx context.Context, cardID int, tagIDs []int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]int, len(tagIDs))
	copy(cp, tagIDs)
	m.cardTags[cardID] = cp
	return nil
}

func (m *cardStoreMock) CopyCard(ctx context.Context, newDeckID int, src domain.Card) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := &domain.Card{
		ID:         m.nextID,
		DeckID:     newDeckID,
		Question:   src.Question,
		Answer:     src.Answer,
		CategoryID: src.CategoryID,
	}
	m.nextID++
	m.cards[c.ID] = c
	return nil
}

func (m *cardStoreMock) GetCardTagIDs(ctx context.Context, cardID int) ([]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids, ok := m.cardTags[cardID]
	if !ok {
		return nil, nil
	}
	cp := make([]int, len(ids))
	copy(cp, ids)
	return cp, nil
}

// ---------------------------------------------------------------------------
// deckStoreMock — реализует DeckStore + CardDeckStore
// ---------------------------------------------------------------------------

type deckStoreMock struct {
	mu       sync.Mutex
	decks    map[int]*domain.Deck
	deckTags map[int][]int
	nextID   int
}

func newDeckStoreMock() *deckStoreMock {
	return &deckStoreMock{
		decks:    make(map[int]*domain.Deck),
		deckTags: make(map[int][]int),
		nextID:   1,
	}
}

func (m *deckStoreMock) Create(ctx context.Context, d *domain.Deck) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	d.ID = m.nextID
	m.nextID++
	d.CreatedAt = time.Now()
	d.UpdatedAt = d.CreatedAt
	cp := *d
	m.decks[d.ID] = &cp
	return nil
}

func (m *deckStoreMock) GetByID(ctx context.Context, id int) (*domain.Deck, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	d, ok := m.decks[id]
	if !ok {
		return nil, nil
	}
	cp := *d
	return &cp, nil
}

func (m *deckStoreMock) ListByUserID(ctx context.Context, userID int) ([]domain.Deck, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Deck
	for _, d := range m.decks {
		if d.UserID == userID {
			list = append(list, *d)
		}
	}
	return list, nil
}

func (m *deckStoreMock) ListByUserIDWithFilters(ctx context.Context, userID int, page, limit int, categoryID *int, search string) ([]domain.Deck, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Deck
	for _, d := range m.decks {
		if d.UserID == userID {
			list = append(list, *d)
		}
	}
	total := len(list)
	offset := (page - 1) * limit
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return list[offset:end], total, nil
}

func (m *deckStoreMock) ListPublic(ctx context.Context, limit, offset int) ([]domain.Deck, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Deck
	for _, d := range m.decks {
		if d.IsPublic {
			list = append(list, *d)
		}
	}
	if offset >= len(list) {
		return nil, nil
	}
	end := offset + limit
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], nil
}

func (m *deckStoreMock) ListPublicWithFilters(ctx context.Context, page, limit int, categoryID *int, search string, sortBy string) ([]domain.Deck, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Deck
	for _, d := range m.decks {
		if d.IsPublic {
			list = append(list, *d)
		}
	}
	total := len(list)
	offset := (page - 1) * limit
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return list[offset:end], total, nil
}

func (m *deckStoreMock) Update(ctx context.Context, d *domain.Deck) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.decks[d.ID]
	if !ok {
		return errors.New("not found")
	}
	d.UpdatedAt = time.Now()
	cp := *d
	m.decks[d.ID] = &cp
	return nil
}

func (m *deckStoreMock) Delete(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.decks, id)
	delete(m.deckTags, id)
	return nil
}

func (m *deckStoreMock) SetDeckTags(ctx context.Context, deckID int, tagIDs []int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]int, len(tagIDs))
	copy(cp, tagIDs)
	m.deckTags[deckID] = cp
	return nil
}

func (m *deckStoreMock) GetDeckTagIDs(ctx context.Context, deckID int) ([]int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ids, ok := m.deckTags[deckID]
	if !ok {
		return nil, nil
	}
	cp := make([]int, len(ids))
	copy(cp, ids)
	return cp, nil
}

// ---------------------------------------------------------------------------
// categoryStoreMock
// ---------------------------------------------------------------------------

type categoryStoreMock struct {
	mu     sync.Mutex
	byID   map[int]*domain.Category
	nextID int
}

func newCategoryStoreMock() *categoryStoreMock {
	return &categoryStoreMock{byID: make(map[int]*domain.Category), nextID: 1}
}

func (m *categoryStoreMock) Create(ctx context.Context, c *domain.Category) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	c.ID = m.nextID
	m.nextID++
	c.CreatedAt = time.Now()
	cp := *c
	m.byID[c.ID] = &cp
	return nil
}

func (m *categoryStoreMock) GetByID(ctx context.Context, id int) (*domain.Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (m *categoryStoreMock) GetByName(ctx context.Context, name string) (*domain.Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.byID {
		if c.Name == name {
			cp := *c
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *categoryStoreMock) List(ctx context.Context) ([]domain.Category, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Category
	for _, c := range m.byID {
		list = append(list, *c)
	}
	return list, nil
}

func (m *categoryStoreMock) Update(ctx context.Context, c *domain.Category) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.byID[c.ID]
	if !ok {
		return errors.New("not found")
	}
	cp := *c
	m.byID[c.ID] = &cp
	return nil
}

func (m *categoryStoreMock) Delete(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byID, id)
	return nil
}

// ---------------------------------------------------------------------------
// tagStoreMock
// ---------------------------------------------------------------------------

type tagStoreMock struct {
	mu     sync.Mutex
	byID   map[int]*domain.Tag
	nextID int
}

func newTagStoreMock() *tagStoreMock {
	return &tagStoreMock{byID: make(map[int]*domain.Tag), nextID: 1}
}

func (m *tagStoreMock) Create(ctx context.Context, t *domain.Tag) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	t.ID = m.nextID
	m.nextID++
	t.CreatedAt = time.Now()
	cp := *t
	m.byID[t.ID] = &cp
	return nil
}

func (m *tagStoreMock) GetByID(ctx context.Context, id int) (*domain.Tag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	t, ok := m.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	return &cp, nil
}

func (m *tagStoreMock) GetByName(ctx context.Context, name string) (*domain.Tag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.byID {
		if t.Name == name {
			cp := *t
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *tagStoreMock) GetByIDs(ctx context.Context, ids []int) ([]domain.Tag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Tag
	for _, id := range ids {
		if t, ok := m.byID[id]; ok {
			list = append(list, *t)
		}
	}
	return list, nil
}

func (m *tagStoreMock) List(ctx context.Context) ([]domain.Tag, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []domain.Tag
	for _, t := range m.byID {
		list = append(list, *t)
	}
	return list, nil
}

func (m *tagStoreMock) ListWithSearch(ctx context.Context, search string) ([]domain.Tag, error) {
	return m.List(ctx)
}

func (m *tagStoreMock) Update(ctx context.Context, t *domain.Tag) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.byID[t.ID]
	if !ok {
		return errors.New("not found")
	}
	cp := *t
	m.byID[t.ID] = &cp
	return nil
}

func (m *tagStoreMock) Delete(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byID, id)
	return nil
}

// ---------------------------------------------------------------------------
// contentUserStoreMock — реализует DeckUserStore
// ---------------------------------------------------------------------------

type contentUserStoreMock struct {
	mu     sync.Mutex
	byID   map[int]*domain.User
	nextID int
}

func newContentUserStoreMock() *contentUserStoreMock {
	return &contentUserStoreMock{byID: make(map[int]*domain.User), nextID: 1}
}

func (m *contentUserStoreMock) GetByID(ctx context.Context, id int) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (m *contentUserStoreMock) seed(id int, username string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	uname := username
	m.byID[id] = &domain.User{ID: id, Email: username + "@test.com", Username: &uname}
}
