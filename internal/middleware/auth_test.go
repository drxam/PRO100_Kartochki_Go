package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/pkg/jwt"
	"github.com/stretchr/testify/require"
)

// userCheckerMock — простейшая реализация UserStatusChecker для тестов middleware.
type userCheckerMock struct {
	mu    sync.Mutex
	users map[int]*domain.User
}

func newUserCheckerMock() *userCheckerMock {
	return &userCheckerMock{users: make(map[int]*domain.User)}
}

func (m *userCheckerMock) add(u *domain.User) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[u.ID] = u
}

// GetByID имитирует UserRepository: возвращает nil для удалённых.
func (m *userCheckerMock) GetByID(_ context.Context, id int) (*domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok || u.DeletedAt != nil {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func newTestJWT(t *testing.T) *jwt.Manager {
	t.Helper()
	return jwt.NewManager(jwt.Config{
		AccessSecret:  "test-access",
		RefreshSecret: "test-refresh",
		AccessTTL:     5 * time.Minute,
		RefreshTTL:    1 * time.Hour,
	})
}

// helper: одну ручку «эхо user_id» под Auth.
func runAuth(t *testing.T, header string, checker UserStatusChecker, jm *jwt.Manager) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Auth(jm, checker))
	r.GET("/x", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": GetUserID(c), "role": GetUserRole(c)})
	})
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAuth_NoHeader_401(t *testing.T) {
	checker := newUserCheckerMock()
	w := runAuth(t, "", checker, newTestJWT(t))
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "UNAUTHORIZED")
}

func TestAuth_BadFormat_401(t *testing.T) {
	checker := newUserCheckerMock()
	w := runAuth(t, "NotBearer abc", checker, newTestJWT(t))
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "INVALID_TOKEN")
}

func TestAuth_InvalidJWT_401(t *testing.T) {
	checker := newUserCheckerMock()
	w := runAuth(t, "Bearer not-a-jwt", checker, newTestJWT(t))
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuth_DeletedUser_401(t *testing.T) {
	checker := newUserCheckerMock()
	jm := newTestJWT(t)
	tok, err := jm.GenerateAccessToken(42, "x@y.z", string(domain.RoleUser), 1)
	require.NoError(t, err)
	// пользователя нет в checker — middleware должен ответить 401.
	w := runAuth(t, "Bearer "+tok, checker, jm)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "учётная запись недоступна")
}

func TestAuth_BlockedUser_403(t *testing.T) {
	checker := newUserCheckerMock()
	checker.add(&domain.User{ID: 7, Email: "blocked@test.com", Role: string(domain.RoleUser), IsBlocked: true, TokenVersion: 1})
	jm := newTestJWT(t)
	tok, _ := jm.GenerateAccessToken(7, "blocked@test.com", string(domain.RoleUser), 1)

	w := runAuth(t, "Bearer "+tok, checker, jm)
	require.Equal(t, http.StatusForbidden, w.Code)
	require.Contains(t, w.Body.String(), "заблокирована")
}

func TestAuth_HappyPath_PassesContext(t *testing.T) {
	checker := newUserCheckerMock()
	checker.add(&domain.User{ID: 1, Email: "ok@test.com", Role: string(domain.RoleAdmin), TokenVersion: 1})
	jm := newTestJWT(t)
	tok, _ := jm.GenerateAccessToken(1, "ok@test.com", string(domain.RoleUser), 1)
	// Замечание: токен выпущен с role=user, но middleware кладёт в контекст
	// АКТУАЛЬНУЮ роль из БД (admin). Это тоже проверяем.

	w := runAuth(t, "Bearer "+tok, checker, jm)
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"user_id":1`)
	require.Contains(t, w.Body.String(), `"role":"admin"`)
}

func TestAuth_RevokedByTokenVersion_401(t *testing.T) {
	checker := newUserCheckerMock()
	checker.add(&domain.User{ID: 1, Email: "ok@test.com", Role: string(domain.RoleUser), TokenVersion: 2})
	jm := newTestJWT(t)
	// Токен выпущен, когда token_version был 1 (старое значение).
	tok, _ := jm.GenerateAccessToken(1, "ok@test.com", string(domain.RoleUser), 1)

	w := runAuth(t, "Bearer "+tok, checker, jm)
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Contains(t, w.Body.String(), "токен отозван")
}

// --- RequireRole -------------------------------------------------------

func runRoleChain(t *testing.T, role domain.UserRole, allowed ...domain.UserRole) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Имитируем то, что Auth уже отработал и положил роль в контекст.
	r.Use(func(c *gin.Context) {
		c.Set(UserIDKey, 1)
		c.Set(UserRoleKey, string(role))
		c.Next()
	})
	r.Use(RequireRole(allowed...))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestRequireRole_Allowed(t *testing.T) {
	require.Equal(t, http.StatusOK,
		runRoleChain(t, domain.RoleAdmin, domain.RoleAdmin))
}

func TestRequireRole_AllowedMultiple(t *testing.T) {
	require.Equal(t, http.StatusOK,
		runRoleChain(t, domain.RoleModerator, domain.RoleAdmin, domain.RoleModerator))
}

func TestRequireRole_Denied(t *testing.T) {
	require.Equal(t, http.StatusForbidden,
		runRoleChain(t, domain.RoleUser, domain.RoleAdmin))
}

func TestRequireRole_NoRoleInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireRole(domain.RoleAdmin))
	r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}
