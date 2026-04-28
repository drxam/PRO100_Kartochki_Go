package service

import (
	"context"
	"testing"
	"time"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/drxam/PRO100_Kartochki_Go/internal/mailer"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// helper: создать AuthService с in-memory моками.
func newTestAuthService() (*AuthService, *userStoreMock, *refreshStoreMock, *resetStoreMock) {
	users := newUserStoreMock()
	tokens := newRefreshStoreMock()
	resets := newResetStoreMock()
	svc := NewAuthService(users, tokens, resets, newTestJWT(),
		&mailer.NoopMailer{}, "http://localhost:8080")
	return svc, users, tokens, resets
}

// helper: вставить готового пользователя с захешированным паролем.
func seedUser(t *testing.T, store *userStoreMock, email, password string) *domain.User {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)
	u := &domain.User{Email: email, PasswordHash: string(hash), Role: string(domain.RoleUser)}
	require.NoError(t, store.Create(context.Background(), u))
	stored, _ := store.GetByID(context.Background(), u.ID)
	return stored
}

func TestAuthService_Register_Success(t *testing.T) {
	svc, users, tokens, _ := newTestAuthService()
	resp, err := svc.Register(context.Background(), domain.RegisterRequest{
		Email: "new@test.com", Password: "secret12",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "new@test.com", resp.User.Email)
	require.Equal(t, string(domain.RoleUser), resp.User.Role)
	require.NotEmpty(t, resp.AccessToken)
	require.NotEmpty(t, resp.RefreshToken)

	// В сторе появился пользователь и записан refresh-токен.
	stored, _ := users.GetByEmail(context.Background(), "new@test.com")
	require.NotNil(t, stored)
	require.Equal(t, 1, tokens.count())
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc, users, _, _ := newTestAuthService()
	seedUser(t, users, "dup@test.com", "secret12")

	_, err := svc.Register(context.Background(), domain.RegisterRequest{
		Email: "dup@test.com", Password: "secret12",
	})
	require.ErrorIs(t, err, ErrEmailExists)
}

func TestAuthService_Login_Success(t *testing.T) {
	svc, users, tokens, _ := newTestAuthService()
	seedUser(t, users, "alice@test.com", "alicepass1")

	resp, err := svc.Login(context.Background(), domain.LoginRequest{
		Email: "alice@test.com", Password: "alicepass1",
	})
	require.NoError(t, err)
	require.Equal(t, "alice@test.com", resp.User.Email)
	require.NotEmpty(t, resp.AccessToken)
	require.Equal(t, 1, tokens.count())
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	svc, users, _, _ := newTestAuthService()
	seedUser(t, users, "alice@test.com", "alicepass1")

	_, err := svc.Login(context.Background(), domain.LoginRequest{
		Email: "alice@test.com", Password: "WRONGPASS",
	})
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_UnknownEmail(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	_, err := svc.Login(context.Background(), domain.LoginRequest{
		Email: "ghost@test.com", Password: "anything1",
	})
	require.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_BlockedUser(t *testing.T) {
	svc, users, _, _ := newTestAuthService()
	u := seedUser(t, users, "blocked@test.com", "secret12")
	require.NoError(t, users.SetBlocked(context.Background(), u.ID, true))

	_, err := svc.Login(context.Background(), domain.LoginRequest{
		Email: "blocked@test.com", Password: "secret12",
	})
	require.ErrorIs(t, err, ErrUserBlocked)
}

func TestAuthService_Refresh_Success(t *testing.T) {
	svc, users, tokens, _ := newTestAuthService()
	seedUser(t, users, "bob@test.com", "secret12")

	loginResp, err := svc.Login(context.Background(), domain.LoginRequest{
		Email: "bob@test.com", Password: "secret12",
	})
	require.NoError(t, err)
	oldRefresh := loginResp.RefreshToken

	refreshResp, err := svc.Refresh(context.Background(), oldRefresh)
	require.NoError(t, err)
	require.NotEmpty(t, refreshResp.AccessToken)
	require.NotEqual(t, oldRefresh, refreshResp.RefreshToken)

	// Старый refresh должен быть удалён, новый — записан.
	got, _ := tokens.GetByToken(context.Background(), oldRefresh)
	require.Nil(t, got)
	got, _ = tokens.GetByToken(context.Background(), refreshResp.RefreshToken)
	require.NotNil(t, got)
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	_, err := svc.Refresh(context.Background(), "not-a-jwt")
	require.ErrorIs(t, err, ErrRefreshTokenInvalid)
}

func TestAuthService_Refresh_BlockedUser(t *testing.T) {
	svc, users, tokens, _ := newTestAuthService()
	seedUser(t, users, "bob@test.com", "secret12")

	loginResp, err := svc.Login(context.Background(), domain.LoginRequest{
		Email: "bob@test.com", Password: "secret12",
	})
	require.NoError(t, err)

	u, _ := users.GetByEmail(context.Background(), "bob@test.com")
	require.NoError(t, users.SetBlocked(context.Background(), u.ID, true))

	_, err = svc.Refresh(context.Background(), loginResp.RefreshToken)
	require.ErrorIs(t, err, ErrUserBlocked)
	// Все токены пользователя должны быть инвалидированы.
	require.Equal(t, 0, tokens.count())
}

func TestAuthService_Logout_DeletesToken(t *testing.T) {
	svc, users, tokens, _ := newTestAuthService()
	seedUser(t, users, "bob@test.com", "secret12")

	loginResp, err := svc.Login(context.Background(), domain.LoginRequest{
		Email: "bob@test.com", Password: "secret12",
	})
	require.NoError(t, err)
	require.NoError(t, svc.Logout(context.Background(), loginResp.RefreshToken))
	require.Equal(t, 0, tokens.count())
}

// --- Восстановление пароля (ТЗ §4.2) -----------------------------------

func TestAuthService_RequestPasswordReset_ExistingUser(t *testing.T) {
	svc, users, _, resets := newTestAuthService()
	seedUser(t, users, "bob@test.com", "secret12")

	tok, ok, err := svc.RequestPasswordReset(context.Background(), "bob@test.com")
	require.NoError(t, err)
	require.True(t, ok)
	require.NotEmpty(t, tok)

	prt, _ := resets.GetByToken(context.Background(), tok)
	require.NotNil(t, prt)
	require.Nil(t, prt.UsedAt)
}

func TestAuthService_RequestPasswordReset_UnknownEmail_OkFalse(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	tok, ok, err := svc.RequestPasswordReset(context.Background(), "nobody@nowhere.com")
	require.NoError(t, err)
	require.False(t, ok)
	require.Empty(t, tok)
}

func TestAuthService_RequestPasswordReset_BlockedUser_OkFalse(t *testing.T) {
	svc, users, _, _ := newTestAuthService()
	u := seedUser(t, users, "blocked@test.com", "secret12")
	require.NoError(t, users.SetBlocked(context.Background(), u.ID, true))

	_, ok, err := svc.RequestPasswordReset(context.Background(), "blocked@test.com")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestAuthService_RequestPasswordReset_InvalidatesPrevious(t *testing.T) {
	svc, users, _, resets := newTestAuthService()
	seedUser(t, users, "bob@test.com", "secret12")

	first, _, err := svc.RequestPasswordReset(context.Background(), "bob@test.com")
	require.NoError(t, err)

	_, _, err = svc.RequestPasswordReset(context.Background(), "bob@test.com")
	require.NoError(t, err)

	prt, _ := resets.GetByToken(context.Background(), first)
	require.NotNil(t, prt.UsedAt, "первый токен должен быть погашен при выпуске нового")
}

func TestAuthService_ResetPassword_Success(t *testing.T) {
	svc, users, tokens, _ := newTestAuthService()
	u := seedUser(t, users, "bob@test.com", "oldpass11")

	tok, _, err := svc.RequestPasswordReset(context.Background(), "bob@test.com")
	require.NoError(t, err)

	// сделаем активный refresh-токен через login
	_, err = svc.Login(context.Background(), domain.LoginRequest{Email: "bob@test.com", Password: "oldpass11"})
	require.NoError(t, err)
	require.Equal(t, 1, tokens.count())

	require.NoError(t, svc.ResetPassword(context.Background(), tok, "newpass11"))

	// Старые refresh-токены инвалидированы.
	require.Equal(t, 0, tokens.count())
	// Старый пароль не работает, новый работает.
	_, err = svc.Login(context.Background(), domain.LoginRequest{Email: "bob@test.com", Password: "oldpass11"})
	require.ErrorIs(t, err, ErrInvalidCredentials)

	got, _ := svc.Login(context.Background(), domain.LoginRequest{Email: "bob@test.com", Password: "newpass11"})
	require.NotNil(t, got)
	_ = u
}

func TestAuthService_ResetPassword_InvalidToken(t *testing.T) {
	svc, _, _, _ := newTestAuthService()
	err := svc.ResetPassword(context.Background(), "GHOST", "newpass11")
	require.ErrorIs(t, err, ErrInvalidResetToken)
}

func TestAuthService_ResetPassword_AlreadyUsed(t *testing.T) {
	svc, users, _, _ := newTestAuthService()
	seedUser(t, users, "bob@test.com", "oldpass11")
	tok, _, err := svc.RequestPasswordReset(context.Background(), "bob@test.com")
	require.NoError(t, err)
	require.NoError(t, svc.ResetPassword(context.Background(), tok, "newpass11"))

	err = svc.ResetPassword(context.Background(), tok, "another1!")
	require.ErrorIs(t, err, ErrInvalidResetToken)
}

func TestAuthService_ResetPassword_Expired(t *testing.T) {
	svc, users, _, resets := newTestAuthService()
	seedUser(t, users, "bob@test.com", "oldpass11")
	tok, _, err := svc.RequestPasswordReset(context.Background(), "bob@test.com")
	require.NoError(t, err)

	// Сдвинем expires_at в прошлое.
	prt, _ := resets.GetByToken(context.Background(), tok)
	resets.byTok[tok].ExpiresAt = time.Now().Add(-1 * time.Minute)
	_ = prt

	err = svc.ResetPassword(context.Background(), tok, "newpass11")
	require.ErrorIs(t, err, ErrResetTokenExpired)
}

func TestAuthService_ResetPassword_BlockedUser(t *testing.T) {
	svc, users, _, _ := newTestAuthService()
	u := seedUser(t, users, "blocked@test.com", "oldpass11")

	// Сначала выдадим токен, потом заблокируем.
	tok, _, err := svc.RequestPasswordReset(context.Background(), "blocked@test.com")
	require.NoError(t, err)
	require.NoError(t, users.SetBlocked(context.Background(), u.ID, true))

	err = svc.ResetPassword(context.Background(), tok, "newpass11")
	require.ErrorIs(t, err, ErrUserBlocked)
}
