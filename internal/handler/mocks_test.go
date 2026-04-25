package handler

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
)

// In-memory моки сторов. Не дублируют service/mocks_test.go (тот пакет-приватный),
// и здесь нужны для интеграционных тестов hander+gin+service+middleware.

type userStoreMock struct {
	mu     sync.Mutex
	byID   map[int]*domain.User
	nextID int
}

func newUserStoreMock() *userStoreMock {
	return &userStoreMock{byID: make(map[int]*domain.User), nextID: 1}
}

func (m *userStoreMock) Create(ctx context.Context, u *domain.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.byID {
		if e.Email == u.Email && e.DeletedAt == nil {
			return errors.New("email exists")
		}
	}
	u.ID = m.nextID
	m.nextID++
	u.CreatedAt = time.Now()
	u.UpdatedAt = u.CreatedAt
	if u.Role == "" {
		u.Role = string(domain.RoleUser)
	}
	if u.TokenVersion == 0 {
		u.TokenVersion = 1
	}
	cp := *u
	m.byID[u.ID] = &cp
	return nil
}
func (m *userStoreMock) GetByID(ctx context.Context, id int) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok || u.DeletedAt != nil {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}
func (m *userStoreMock) GetByIDIncludingDeleted(ctx context.Context, id int) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}
func (m *userStoreMock) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.byID {
		if u.Email == email && u.DeletedAt == nil {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}
func (m *userStoreMock) UpdatePassword(ctx context.Context, id int, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return errors.New("not found")
	}
	u.PasswordHash = hash
	return nil
}
func (m *userStoreMock) IncrementTokenVersion(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok {
		return errors.New("not found")
	}
	u.TokenVersion++
	return nil
}
func (m *userStoreMock) List(ctx context.Context, page, limit int, includeDeleted bool) ([]domain.User, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.User, 0, len(m.byID))
	for _, u := range m.byID {
		if !includeDeleted && u.DeletedAt != nil {
			continue
		}
		out = append(out, *u)
	}
	return out, len(out), nil
}
func (m *userStoreMock) SetBlocked(ctx context.Context, id int, blocked bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok || u.DeletedAt != nil {
		return errors.New("not found")
	}
	u.IsBlocked = blocked
	if blocked {
		now := time.Now()
		u.BlockedAt = &now
	} else {
		u.BlockedAt = nil
	}
	return nil
}
func (m *userStoreMock) SoftDelete(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok || u.DeletedAt != nil {
		return errors.New("not found")
	}
	now := time.Now()
	u.DeletedAt = &now
	return nil
}
func (m *userStoreMock) SetRole(ctx context.Context, id int, role string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.byID[id]
	if !ok || u.DeletedAt != nil {
		return errors.New("not found")
	}
	u.Role = role
	return nil
}

// ----------------------------------------------------------------------

type refreshStoreMock struct {
	mu    sync.Mutex
	byTok map[string]*domain.RefreshToken
	id    int
}

func newRefreshStoreMock() *refreshStoreMock {
	return &refreshStoreMock{byTok: make(map[string]*domain.RefreshToken), id: 1}
}
func (m *refreshStoreMock) Create(ctx context.Context, rt *domain.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byTok[rt.Token]; ok {
		return errors.New("dup")
	}
	rt.ID = m.id
	m.id++
	rt.CreatedAt = time.Now()
	cp := *rt
	m.byTok[rt.Token] = &cp
	return nil
}
func (m *refreshStoreMock) GetByToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rt, ok := m.byTok[token]
	if !ok {
		return nil, nil
	}
	cp := *rt
	return &cp, nil
}
func (m *refreshStoreMock) DeleteByToken(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.byTok, token)
	return nil
}
func (m *refreshStoreMock) DeleteByUserID(ctx context.Context, userID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range m.byTok {
		if v.UserID == userID {
			delete(m.byTok, k)
		}
	}
	return nil
}

// ----------------------------------------------------------------------

type resetStoreMock struct {
	mu    sync.Mutex
	byTok map[string]*domain.PasswordResetToken
	id    int
}

func newResetStoreMock() *resetStoreMock {
	return &resetStoreMock{byTok: make(map[string]*domain.PasswordResetToken), id: 1}
}
func (m *resetStoreMock) Create(ctx context.Context, prt *domain.PasswordResetToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	prt.ID = m.id
	m.id++
	prt.CreatedAt = time.Now()
	cp := *prt
	m.byTok[prt.Token] = &cp
	return nil
}
func (m *resetStoreMock) GetByToken(ctx context.Context, token string) (*domain.PasswordResetToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prt, ok := m.byTok[token]
	if !ok {
		return nil, nil
	}
	cp := *prt
	return &cp, nil
}
func (m *resetStoreMock) MarkUsed(ctx context.Context, id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, prt := range m.byTok {
		if prt.ID == id {
			now := time.Now()
			prt.UsedAt = &now
			return nil
		}
	}
	return errors.New("not found")
}
func (m *resetStoreMock) InvalidateActiveForUser(ctx context.Context, userID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for _, prt := range m.byTok {
		if prt.UserID == userID && prt.UsedAt == nil {
			prt.UsedAt = &now
		}
	}
	return nil
}
