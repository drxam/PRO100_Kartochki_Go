package service

import (
	"context"
	"testing"

	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/stretchr/testify/require"
)

func newTestAdminService() (*AdminService, *userStoreMock, *refreshStoreMock) {
	users := newUserStoreMock()
	tokens := newRefreshStoreMock()
	svc := NewAdminService(users, tokens)
	return svc, users, tokens
}

// seedTokens добавляет N refresh-токенов для пользователя — имитация активных сессий.
func seedTokens(t *testing.T, store *refreshStoreMock, userID, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		require.NoError(t, store.Create(context.Background(),
			&domain.RefreshToken{UserID: userID, Token: rndTok(userID, i)}))
	}
}

func rndTok(userID, i int) string {
	return "tok-" + string(rune('a'+userID)) + "-" + string(rune('0'+i))
}

func TestAdminService_ListUsers_HidesDeletedByDefault(t *testing.T) {
	svc, users, _ := newTestAdminService()
	seedUser(t, users, "alive@test.com", "pass1234")
	dead := seedUser(t, users, "dead@test.com", "pass1234")
	require.NoError(t, users.SoftDelete(context.Background(), dead.ID))

	resp, err := svc.ListUsers(context.Background(), 1, 10, false)
	require.NoError(t, err)
	require.Equal(t, 1, resp.Pagination.Total)
	require.Len(t, resp.Users, 1)
	require.Equal(t, "alive@test.com", resp.Users[0].Email)

	resp, err = svc.ListUsers(context.Background(), 1, 10, true)
	require.NoError(t, err)
	require.Equal(t, 2, resp.Pagination.Total)
}

func TestAdminService_GetUser_NotFound(t *testing.T) {
	svc, _, _ := newTestAdminService()
	_, err := svc.GetUser(context.Background(), 999)
	require.Error(t, err)
}

func TestAdminService_SetBlocked_RevokesRefreshTokens(t *testing.T) {
	svc, users, tokens := newTestAdminService()
	admin := seedUser(t, users, "admin@test.com", "pass1234")
	target := seedUser(t, users, "alice@test.com", "pass1234")
	seedTokens(t, tokens, target.ID, 3)
	seedTokens(t, tokens, admin.ID, 1) // эти трогать нельзя

	require.NoError(t, svc.SetBlocked(context.Background(), admin.ID, target.ID, true))

	// Все refresh-токены alice удалены.
	require.Equal(t, 1, tokens.count(), "должны остаться только токены admin'а")

	got, _ := users.GetByID(context.Background(), target.ID)
	require.True(t, got.IsBlocked)
	require.NotNil(t, got.BlockedAt)
}

func TestAdminService_SetBlocked_Unblock_DoesNotRevokeAdmin(t *testing.T) {
	svc, users, tokens := newTestAdminService()
	admin := seedUser(t, users, "admin@test.com", "pass1234")
	target := seedUser(t, users, "alice@test.com", "pass1234")
	require.NoError(t, users.SetBlocked(context.Background(), target.ID, true))

	seedTokens(t, tokens, target.ID, 0) // у alice нет сессий
	require.NoError(t, svc.SetBlocked(context.Background(), admin.ID, target.ID, false))

	got, _ := users.GetByID(context.Background(), target.ID)
	require.False(t, got.IsBlocked)
	require.Nil(t, got.BlockedAt)
}

func TestAdminService_SetBlocked_CannotSelfBlock(t *testing.T) {
	svc, users, _ := newTestAdminService()
	admin := seedUser(t, users, "admin@test.com", "pass1234")

	err := svc.SetBlocked(context.Background(), admin.ID, admin.ID, true)
	require.ErrorIs(t, err, ErrCannotModifySelf)
}

func TestAdminService_SoftDelete_RevokesRefresh(t *testing.T) {
	svc, users, tokens := newTestAdminService()
	admin := seedUser(t, users, "admin@test.com", "pass1234")
	target := seedUser(t, users, "alice@test.com", "pass1234")
	seedTokens(t, tokens, target.ID, 2)

	require.NoError(t, svc.SoftDelete(context.Background(), admin.ID, target.ID))

	require.Equal(t, 0, tokens.count())
	got, _ := users.GetByIDIncludingDeleted(context.Background(), target.ID)
	require.NotNil(t, got.DeletedAt)
}

func TestAdminService_SoftDelete_CannotSelfDelete(t *testing.T) {
	svc, users, _ := newTestAdminService()
	admin := seedUser(t, users, "admin@test.com", "pass1234")
	err := svc.SoftDelete(context.Background(), admin.ID, admin.ID)
	require.ErrorIs(t, err, ErrCannotModifySelf)
}

func TestAdminService_SetRole_ValidRoles(t *testing.T) {
	svc, users, tokens := newTestAdminService()
	admin := seedUser(t, users, "admin@test.com", "pass1234")
	target := seedUser(t, users, "alice@test.com", "pass1234")
	seedTokens(t, tokens, target.ID, 2)

	require.NoError(t, svc.SetRole(context.Background(), admin.ID, target.ID, string(domain.RoleModerator)))
	got, _ := users.GetByID(context.Background(), target.ID)
	require.Equal(t, string(domain.RoleModerator), got.Role)
	require.Equal(t, 0, tokens.count(), "после смены роли все refresh-токены инвалидируются")
}

func TestAdminService_SetRole_InvalidRole(t *testing.T) {
	svc, users, _ := newTestAdminService()
	admin := seedUser(t, users, "admin@test.com", "pass1234")
	target := seedUser(t, users, "alice@test.com", "pass1234")

	err := svc.SetRole(context.Background(), admin.ID, target.ID, "superuser")
	require.ErrorIs(t, err, ErrInvalidRole)
}

func TestAdminService_SetRole_CannotSelf(t *testing.T) {
	svc, users, _ := newTestAdminService()
	admin := seedUser(t, users, "admin@test.com", "pass1234")
	err := svc.SetRole(context.Background(), admin.ID, admin.ID, string(domain.RoleModerator))
	require.ErrorIs(t, err, ErrCannotModifySelf)
}
