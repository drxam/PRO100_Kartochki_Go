package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/jwt"
)

// Минимальные in-memory моки UserStore / RefreshTokenStore / PasswordResetStore
// для unit-тестов сервисов. Не потокобезопасны иначе чем нужно для одного теста.

// ----- UserStore mock ---------------------------------------------------

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
	for _, existing := range m.byID {
		if existing.Email == u.Email && existing.DeletedAt == nil {
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
		u.TokenVersion = 1 // соответствует DEFAULT 1 в схеме
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
	u.UpdatedAt = time.Now()
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
	u.UpdatedAt = time.Now()
	return nil
}

// AdminUserStore methods.

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

// ----- RefreshTokenStore mock ------------------------------------------

type refreshStoreMock struct {
	mu     sync.Mutex
	byTok  map[string]*domain.RefreshToken
	nextID int
}

func newRefreshStoreMock() *refreshStoreMock {
	return &refreshStoreMock{byTok: make(map[string]*domain.RefreshToken), nextID: 1}
}

func (m *refreshStoreMock) Create(ctx context.Context, rt *domain.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byTok[rt.Token]; exists {
		return errors.New("duplicate token")
	}
	rt.ID = m.nextID
	m.nextID++
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
	for tok, rt := range m.byTok {
		if rt.UserID == userID {
			delete(m.byTok, tok)
		}
	}
	return nil
}

func (m *refreshStoreMock) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.byTok)
}

// ----- PasswordResetStore mock -----------------------------------------

type resetStoreMock struct {
	mu     sync.Mutex
	byTok  map[string]*domain.PasswordResetToken
	nextID int
}

func newResetStoreMock() *resetStoreMock {
	return &resetStoreMock{byTok: make(map[string]*domain.PasswordResetToken), nextID: 1}
}

func (m *resetStoreMock) Create(ctx context.Context, prt *domain.PasswordResetToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	prt.ID = m.nextID
	m.nextID++
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

// ----- newJWTManager helper --------------------------------------------

func newTestJWT() *jwt.Manager {
	return jwt.NewManager(jwt.Config{
		AccessSecret:  "test-access",
		RefreshSecret: "test-refresh",
		AccessTTL:     15 * time.Minute,
		RefreshTTL:    24 * time.Hour,
	})
}
