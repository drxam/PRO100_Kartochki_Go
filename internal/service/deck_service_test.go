package service

import (
	"context"
	"testing"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/stretchr/testify/require"
)

func newTestDeckService() (*DeckService, *deckStoreMock, *cardStoreMock, *contentUserStoreMock, *categoryStoreMock, *tagStoreMock) {
	decks := newDeckStoreMock()
	cards := newCardStoreMock()
	users := newContentUserStoreMock()
	cats := newCategoryStoreMock()
	tags := newTagStoreMock()
	svc := NewDeckService(decks, cards, users, cats, tags)
	return svc, decks, cards, users, cats, tags
}

// ---- Create ----------------------------------------------------------------

func TestDeckService_Create_Success(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()

	deck, err := svc.Create(context.Background(), 1, domain.CreateDeckRequest{
		Title:    "Алгоритмы",
		IsPublic: false,
	})
	require.NoError(t, err)
	require.NotNil(t, deck)
	require.Equal(t, "Алгоритмы", deck.Title)
	require.Equal(t, 1, deck.UserID)
	require.False(t, deck.IsPublic)
	require.Equal(t, 1, deck.ID)
}

func TestDeckService_Create_Public(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()

	deck, err := svc.Create(context.Background(), 2, domain.CreateDeckRequest{
		Title:    "Публичный набор",
		IsPublic: true,
	})
	require.NoError(t, err)
	require.True(t, deck.IsPublic)
}

func TestDeckService_Create_WithTags(t *testing.T) {
	svc, _, _, _, _, tags := newTestDeckService()
	tag := &domain.Tag{Name: "go"}
	require.NoError(t, tags.Create(context.Background(), tag))

	deck, err := svc.Create(context.Background(), 1, domain.CreateDeckRequest{
		Title:  "Go набор",
		TagIDs: []int{tag.ID},
	})
	require.NoError(t, err)
	require.Len(t, deck.Tags, 1)
	require.Equal(t, "go", deck.Tags[0].Name)
}

// ---- GetByID ---------------------------------------------------------------

func TestDeckService_GetByID_Success(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	created, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "Набор"})

	found, err := svc.GetByID(context.Background(), created.ID, 1)
	require.NoError(t, err)
	require.Equal(t, "Набор", found.Title)
}

func TestDeckService_GetByID_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	_, err := svc.GetByID(context.Background(), 999, 1)
	require.ErrorIs(t, err, ErrDeckNotFound)
}

func TestDeckService_GetByID_Forbidden_PrivateDeck(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "Приватный", IsPublic: false})

	_, err := svc.GetByID(context.Background(), deck.ID, 2) // другой пользователь
	require.ErrorIs(t, err, ErrDeckForbidden)
}

func TestDeckService_GetByID_PublicDeck_AllowOtherUser(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "Публичный", IsPublic: true})

	found, err := svc.GetByID(context.Background(), deck.ID, 99)
	require.NoError(t, err)
	require.Equal(t, deck.ID, found.ID)
}

// ---- Update ----------------------------------------------------------------

func TestDeckService_Update_Success(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "Старый"})

	newTitle := "Новый"
	updated, err := svc.Update(context.Background(), deck.ID, 1, domain.UpdateDeckRequest{Title: &newTitle})
	require.NoError(t, err)
	require.Equal(t, "Новый", updated.Title)
}

func TestDeckService_Update_ChangeVisibility(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "T", IsPublic: false})

	isPublic := true
	updated, err := svc.Update(context.Background(), deck.ID, 1, domain.UpdateDeckRequest{IsPublic: &isPublic})
	require.NoError(t, err)
	require.True(t, updated.IsPublic)
}

func TestDeckService_Update_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	title := "T"
	_, err := svc.Update(context.Background(), 999, 1, domain.UpdateDeckRequest{Title: &title})
	require.ErrorIs(t, err, ErrDeckNotFound)
}

func TestDeckService_Update_Forbidden(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "T"})

	title := "New"
	_, err := svc.Update(context.Background(), deck.ID, 2, domain.UpdateDeckRequest{Title: &title})
	require.ErrorIs(t, err, ErrDeckForbidden)
}

// ---- Delete ----------------------------------------------------------------

func TestDeckService_Delete_Success(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "Удали"})

	err := svc.Delete(context.Background(), deck.ID, 1)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), deck.ID, 1)
	require.ErrorIs(t, err, ErrDeckNotFound)
}

func TestDeckService_Delete_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	err := svc.Delete(context.Background(), 999, 1)
	require.ErrorIs(t, err, ErrDeckNotFound)
}

func TestDeckService_Delete_Forbidden(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "T"})

	err := svc.Delete(context.Background(), deck.ID, 2) // другой пользователь
	require.ErrorIs(t, err, ErrDeckForbidden)
}

// ---- DeleteAny (admin) -----------------------------------------------------

func TestDeckService_DeleteAny_Success(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "T"})

	err := svc.DeleteAny(context.Background(), deck.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), deck.ID, 1)
	require.ErrorIs(t, err, ErrDeckNotFound)
}

func TestDeckService_DeleteAny_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	err := svc.DeleteAny(context.Background(), 999)
	require.ErrorIs(t, err, ErrDeckNotFound)
}

// ---- ListByUser ------------------------------------------------------------

func TestDeckService_ListByUser_Empty(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	list, err := svc.ListByUser(context.Background(), 1)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestDeckService_ListByUser_OnlyOwnDecks(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	_, _ = svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "User1 deck"})
	_, _ = svc.Create(context.Background(), 2, domain.CreateDeckRequest{Title: "User2 deck"})

	list, err := svc.ListByUser(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "User1 deck", list[0].Title)
}

// ---- GetPublicByID ---------------------------------------------------------

func TestDeckService_GetPublicByID_Success(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "Публичный", IsPublic: true})

	detail, err := svc.GetPublicByID(context.Background(), deck.ID)
	require.NoError(t, err)
	require.Equal(t, deck.ID, detail.ID)
}

func TestDeckService_GetPublicByID_PrivateDeck_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	deck, _ := svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "Приватный", IsPublic: false})

	_, err := svc.GetPublicByID(context.Background(), deck.ID)
	require.ErrorIs(t, err, ErrDeckNotFound)
}

func TestDeckService_GetPublicByID_NotFound(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	_, err := svc.GetPublicByID(context.Background(), 999)
	require.ErrorIs(t, err, ErrDeckNotFound)
}

// ---- ListPublicPaginated ---------------------------------------------------

func TestDeckService_ListPublicPaginated_Success(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	_, _ = svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "Pub1", IsPublic: true})
	_, _ = svc.Create(context.Background(), 1, domain.CreateDeckRequest{Title: "Priv", IsPublic: false})
	_, _ = svc.Create(context.Background(), 2, domain.CreateDeckRequest{Title: "Pub2", IsPublic: true})

	resp, err := svc.ListPublicPaginated(context.Background(), 1, 10, nil, "", "recent")
	require.NoError(t, err)
	require.Equal(t, 2, resp.Pagination.Total)
	require.Len(t, resp.Decks, 2)
}

func TestDeckService_ListPublicPaginated_Pagination(t *testing.T) {
	svc, _, _, _, _, _ := newTestDeckService()
	for i := 0; i < 5; i++ {
		_, _ = svc.Create(context.Background(), 1, domain.CreateDeckRequest{
			Title:    "Deck",
			IsPublic: true,
		})
	}

	resp, err := svc.ListPublicPaginated(context.Background(), 1, 2, nil, "", "recent")
	require.NoError(t, err)
	require.Equal(t, 5, resp.Pagination.Total)
	require.Len(t, resp.Decks, 2)
	require.Equal(t, 1, resp.Pagination.Page)
	require.Equal(t, 2, resp.Pagination.Limit)
}
