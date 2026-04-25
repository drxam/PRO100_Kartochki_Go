package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/middleware"
	"github.com/drxam/PRO100_Kartochki_Go/internal/service"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/jwt"
	"github.com/drxam/PRO100_Kartochki_Go/pkg/validator"
	"github.com/stretchr/testify/require"
)

// testEnv — собранный gin engine + моки сторов + jwt-менеджер; всё, что нужно
// для имитации полного API в тестах handler-слоя.
type testEnv struct {
	r       *gin.Engine
	users   *userStoreMock
	tokens  *refreshStoreMock
	resets  *resetStoreMock
	jwtMgr  *jwt.Manager
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	gin.SetMode(gin.TestMode)

	users := newUserStoreMock()
	tokens := newRefreshStoreMock()
	resets := newResetStoreMock()

	jm := jwt.NewManager(jwt.Config{
		AccessSecret:  "test-access",
		RefreshSecret: "test-refresh",
		AccessTTL:     5 * time.Minute,
		RefreshTTL:    1 * time.Hour,
	})

	authSvc := service.NewAuthService(users, tokens, resets, jm)
	adminSvc := service.NewAdminService(users, tokens)
	v := validator.New()

	authH := NewAuthHandler(authSvc, v)
	authH.SetDevReturnResetToken(true) // удобно для тестов
	adminH := NewAdminHandler(adminSvc, v)

	r := gin.New()
	r.Use(middleware.RequestID())
	api := r.Group("/api")
	{
		api.POST("/auth/register", authH.Register)
		api.POST("/auth/login", authH.Login)
		api.POST("/auth/refresh", authH.Refresh)
		api.POST("/auth/forgot-password", authH.ForgotPassword)
		api.POST("/auth/reset-password", authH.ResetPassword)

		auth := api.Group("")
		auth.Use(middleware.Auth(jm, users))
		{
			auth.POST("/auth/logout", authH.Logout)

			admin := auth.Group("/admin")
			admin.Use(middleware.RequireRole(domain.RoleAdmin))
			{
				admin.GET("/users", adminH.ListUsers)
				admin.GET("/users/:id", adminH.GetUser)
				admin.PATCH("/users/:id/block", adminH.BlockUser)
				admin.PATCH("/users/:id/role", adminH.SetUserRole)
				admin.DELETE("/users/:id", adminH.DeleteUser)
			}
		}
	}
	return &testEnv{r: r, users: users, tokens: tokens, resets: resets, jwtMgr: jm}
}

// do — короткий вызов: метод, путь, тело (или nil), bearer-токен (или "").
// Возвращает recorder для проверки кода и тела.
func (te *testEnv) do(t *testing.T, method, path string, body any, bearer string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		require.NoError(t, err)
		rdr = bytes.NewReader(buf)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	w := httptest.NewRecorder()
	te.r.ServeHTTP(w, req)
	return w
}

// makeAdmin создаёт пользователя с ролью admin и возвращает его id и валидный access-токен.
func (te *testEnv) makeAdmin(t *testing.T) (id int, token string) {
	t.Helper()
	u := &domain.User{Email: "admin@test.com", PasswordHash: "fake", Role: string(domain.RoleAdmin)}
	require.NoError(t, te.users.Create(context.Background(), u))
	tok, err := te.jwtMgr.GenerateAccessToken(u.ID, u.Email, u.Role, 1)
	require.NoError(t, err)
	return u.ID, tok
}

// --- /auth/register --------------------------------------------------------

func TestHandler_Register_Success(t *testing.T) {
	te := newTestEnv(t)
	w := te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "u@test.com", "password": "secret12"}, "")
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var resp domain.AuthRegisterResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.AccessToken)
	require.Equal(t, "u@test.com", resp.User.Email)
}

func TestHandler_Register_DuplicateEmail_409(t *testing.T) {
	te := newTestEnv(t)
	body := map[string]string{"email": "u@test.com", "password": "secret12"}
	te.do(t, http.MethodPost, "/api/auth/register", body, "")
	w := te.do(t, http.MethodPost, "/api/auth/register", body, "")
	require.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Register_BadEmail_400(t *testing.T) {
	te := newTestEnv(t)
	w := te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "not-an-email", "password": "secret12"}, "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Register_WeakPassword_400(t *testing.T) {
	te := newTestEnv(t)
	w := te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "u@test.com", "password": "shortx"}, "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// --- /auth/login -----------------------------------------------------------

func TestHandler_Login_Success(t *testing.T) {
	te := newTestEnv(t)
	te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "u@test.com", "password": "secret12"}, "")
	w := te.do(t, http.MethodPost, "/api/auth/login",
		map[string]string{"email": "u@test.com", "password": "secret12"}, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestHandler_Login_WrongPassword_401(t *testing.T) {
	te := newTestEnv(t)
	te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "u@test.com", "password": "secret12"}, "")
	w := te.do(t, http.MethodPost, "/api/auth/login",
		map[string]string{"email": "u@test.com", "password": "wrong123x"}, "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Login_BlockedUser_403(t *testing.T) {
	te := newTestEnv(t)
	resp := te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "blk@test.com", "password": "secret12"}, "")
	var reg domain.AuthRegisterResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &reg))
	require.NoError(t, te.users.SetBlocked(context.Background(), reg.User.ID, true))

	w := te.do(t, http.MethodPost, "/api/auth/login",
		map[string]string{"email": "blk@test.com", "password": "secret12"}, "")
	require.Equal(t, http.StatusForbidden, w.Code)
}

// --- /auth/refresh + /auth/logout -----------------------------------------

func TestHandler_Refresh_RotatesToken(t *testing.T) {
	te := newTestEnv(t)
	resp := te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "u@test.com", "password": "secret12"}, "")
	var reg domain.AuthRegisterResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &reg))

	w := te.do(t, http.MethodPost, "/api/auth/refresh",
		map[string]string{"refresh_token": reg.RefreshToken}, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var rotated domain.AuthRefreshResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &rotated))
	require.NotEqual(t, reg.RefreshToken, rotated.RefreshToken)
}

func TestHandler_Refresh_InvalidToken_401(t *testing.T) {
	te := newTestEnv(t)
	w := te.do(t, http.MethodPost, "/api/auth/refresh",
		map[string]string{"refresh_token": "garbage"}, "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

// --- /auth/forgot-password + /auth/reset-password -------------------------

func TestHandler_ForgotPassword_KnownAndUnknownLookSame(t *testing.T) {
	te := newTestEnv(t)
	te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "u@test.com", "password": "secret12"}, "")

	resp1 := te.do(t, http.MethodPost, "/api/auth/forgot-password",
		map[string]string{"email": "u@test.com"}, "")
	require.Equal(t, http.StatusOK, resp1.Code)

	resp2 := te.do(t, http.MethodPost, "/api/auth/forgot-password",
		map[string]string{"email": "ghost@nowhere.com"}, "")
	require.Equal(t, http.StatusOK, resp2.Code)

	// Текст одинаковый (защита от account enumeration); токен в ответе только
	// у существующего email (благодаря dev-флагу).
	var r1, r2 domain.ForgotPasswordResponse
	require.NoError(t, json.Unmarshal(resp1.Body.Bytes(), &r1))
	require.NoError(t, json.Unmarshal(resp2.Body.Bytes(), &r2))
	require.Equal(t, r1.Message, r2.Message)
	require.NotEmpty(t, r1.ResetToken)
	require.Empty(t, r2.ResetToken)
}

func TestHandler_ResetPassword_Success(t *testing.T) {
	te := newTestEnv(t)
	te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "u@test.com", "password": "secret12"}, "")
	resp := te.do(t, http.MethodPost, "/api/auth/forgot-password",
		map[string]string{"email": "u@test.com"}, "")
	var r domain.ForgotPasswordResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &r))

	w := te.do(t, http.MethodPost, "/api/auth/reset-password",
		map[string]any{"token": r.ResetToken, "new_password": "newpass11"}, "")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestHandler_ResetPassword_InvalidToken_400(t *testing.T) {
	te := newTestEnv(t)
	w := te.do(t, http.MethodPost, "/api/auth/reset-password",
		map[string]any{"token": "GARBAGE", "new_password": "newpass11"}, "")
	require.Equal(t, http.StatusBadRequest, w.Code)
}

// --- /admin/* (RBAC) ------------------------------------------------------

func TestHandler_Admin_NoToken_401(t *testing.T) {
	te := newTestEnv(t)
	w := te.do(t, http.MethodGet, "/api/admin/users", nil, "")
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandler_Admin_NonAdminUser_403(t *testing.T) {
	te := newTestEnv(t)
	resp := te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "u@test.com", "password": "secret12"}, "")
	var reg domain.AuthRegisterResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &reg))

	w := te.do(t, http.MethodGet, "/api/admin/users", nil, reg.AccessToken)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestHandler_Admin_ListUsers_200(t *testing.T) {
	te := newTestEnv(t)
	_, token := te.makeAdmin(t)

	w := te.do(t, http.MethodGet, "/api/admin/users", nil, token)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp domain.AdminUsersListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.GreaterOrEqual(t, resp.Pagination.Total, 1)
}

func TestHandler_Admin_BlockUser_204(t *testing.T) {
	te := newTestEnv(t)
	_, token := te.makeAdmin(t)
	resp := te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "victim@test.com", "password": "secret12"}, "")
	var reg domain.AuthRegisterResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &reg))

	w := te.do(t, http.MethodPatch, "/api/admin/users/"+strconv.Itoa(reg.User.ID)+"/block",
		map[string]bool{"blocked": true}, token)
	require.Equal(t, http.StatusNoContent, w.Code, w.Body.String())

	got, _ := te.users.GetByID(context.Background(), reg.User.ID)
	require.True(t, got.IsBlocked)
}

func TestHandler_Admin_SetRole_InvalidRole_400(t *testing.T) {
	te := newTestEnv(t)
	_, token := te.makeAdmin(t)
	resp := te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "v@test.com", "password": "secret12"}, "")
	var reg domain.AuthRegisterResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &reg))

	w := te.do(t, http.MethodPatch, "/api/admin/users/"+strconv.Itoa(reg.User.ID)+"/role",
		map[string]string{"role": "godmode"}, token)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Admin_CannotSelfDelete_403(t *testing.T) {
	te := newTestEnv(t)
	id, token := te.makeAdmin(t)

	w := te.do(t, http.MethodDelete, "/api/admin/users/"+strconv.Itoa(id), nil, token)
	require.Equal(t, http.StatusForbidden, w.Code)
}

// --- /auth/logout ---------------------------------------------------------

func TestHandler_Logout_RemovesRefreshToken(t *testing.T) {
	te := newTestEnv(t)
	resp := te.do(t, http.MethodPost, "/api/auth/register",
		map[string]string{"email": "u@test.com", "password": "secret12"}, "")
	var reg domain.AuthRegisterResponse
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &reg))

	w := te.do(t, http.MethodPost, "/api/auth/logout",
		map[string]string{"refresh_token": reg.RefreshToken}, reg.AccessToken)
	require.Equal(t, http.StatusOK, w.Code)

	got, _ := te.tokens.GetByToken(context.Background(), reg.RefreshToken)
	require.Nil(t, got)
}
