package service

import (
	"context"
	"testing"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/stretchr/testify/require"
)

// helpers

func newTestCardService() (*CardService, *cardStoreMock, *deckStoreMock, *categoryStoreMock, *tagStoreMock) {
	cards := newCardStoreMock()
	decks := newDeckStoreMock()
	cats := newCategoryStoreMock()
	tags := newTagStoreMock()
	svc := NewCardService(cards, decks, cats, tags)
	return svc, cards, decks, cats, tags
}

func seedDeck(t *testing.T, store *deckStoreMock, userID int, isPublic bool) *domain.Deck {
	t.Helper()
	d := &domain.Deck{UserID: userID, Title: "Test deck", IsPublic: isPublic}
	_ = store.Create(context.Background(), d)
	return d
}

// ---- Create ----------------------------------------------------------------

func TestCardService_Create_Success(t *testing.T) {
	svc, _, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false)

	card, err := svc.Create(context.Background(), deck.ID, 1, domain.CreateCardRequest{
		Question: "Что такое горутина?",
		Answer:   "Лёгковесный поток в Go",
	})
	require.NoError(t, err)
	require.NotNil(t, card)
	require.Equal(t, "Что такое горутина?", card.Question)
	require.Equal(t, deck.ID, card.DeckID)
}

func TestCardService_Create_DeckIDFromBody(t *testing.T) {
	svc, _, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 2, false)

	deckID := deck.ID
	card, err := svc.Create(context.Background(), 0, 2, domain.CreateCardRequest{
		DeckID:   &deckID,
		Question: "Q",
		Answer:   "A",
	})
	require.NoError(t, err)
	require.Equal(t, deck.ID, card.DeckID)
}

func TestCardService_Create_DeckNotFound(t *testing.T) {
	svc, _, _, _, _ := newTestCardService()

	_, err := svc.Create(context.Background(), 999, 1, domain.CreateCardRequest{
		Question: "Q",
		Answer:   "A",
	})
	require.ErrorIs(t, err, ErrCardNotFound)
}

func TestCardService_Create_Forbidden_WrongOwner(t *testing.T) {
	svc, _, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false) // owner is user 1

	_, err := svc.Create(context.Background(), deck.ID, 2, domain.CreateCardRequest{ // user 2 tries to add
		Question: "Q",
		Answer:   "A",
	})
	require.ErrorIs(t, err, ErrCardForbidden)
}

// ---- GetByID ---------------------------------------------------------------

func TestCardService_GetByID_Success(t *testing.T) {
	svc, cards, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false)
	c := &domain.Card{DeckID: deck.ID, Question: "Q", Answer: "A"}
	_ = cards.Create(context.Background(), c)

	found, err := svc.GetByID(context.Background(), c.ID, 1)
	require.NoError(t, err)
	require.Equal(t, "Q", found.Question)
}

func TestCardService_GetByID_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestCardService()
	_, err := svc.GetByID(context.Background(), 999, 1)
	require.ErrorIs(t, err, ErrCardNotFound)
}

func TestCardService_GetByID_Forbidden_PrivateDeck(t *testing.T) {
	svc, cards, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false) // private deck, owner user 1
	c := &domain.Card{DeckID: deck.ID, Question: "Q", Answer: "A"}
	_ = cards.Create(context.Background(), c)

	_, err := svc.GetByID(context.Background(), c.ID, 2) // user 2 requests
	require.ErrorIs(t, err, ErrCardForbidden)
}

func TestCardService_GetByID_PublicDeck_AllowOtherUser(t *testing.T) {
	svc, cards, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, true) // public deck
	c := &domain.Card{DeckID: deck.ID, Question: "Q", Answer: "A"}
	_ = cards.Create(context.Background(), c)

	found, err := svc.GetByID(context.Background(), c.ID, 99) // any user
	require.NoError(t, err)
	require.NotNil(t, found)
}

// ---- ListByDeck ------------------------------------------------------------

func TestCardService_ListByDeck_Success(t *testing.T) {
	svc, cards, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false)
	_ = cards.Create(context.Background(), &domain.Card{DeckID: deck.ID, Question: "Q1", Answer: "A1"})
	_ = cards.Create(context.Background(), &domain.Card{DeckID: deck.ID, Question: "Q2", Answer: "A2"})

	list, err := svc.ListByDeck(context.Background(), deck.ID, 1)
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestCardService_ListByDeck_EmptyDeck(t *testing.T) {
	svc, _, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false)

	list, err := svc.ListByDeck(context.Background(), deck.ID, 1)
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestCardService_ListByDeck_Forbidden_PrivateDeck(t *testing.T) {
	svc, _, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false)

	_, err := svc.ListByDeck(context.Background(), deck.ID, 2)
	require.ErrorIs(t, err, ErrCardForbidden)
}

// ---- Update ----------------------------------------------------------------

func TestCardService_Update_Success(t *testing.T) {
	svc, cards, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false)
	c := &domain.Card{DeckID: deck.ID, Question: "Old Q", Answer: "Old A"}
	_ = cards.Create(context.Background(), c)

	newQ := "New Q"
	updated, err := svc.Update(context.Background(), c.ID, 1, domain.UpdateCardRequest{Question: &newQ})
	require.NoError(t, err)
	require.Equal(t, "New Q", updated.Question)
	require.Equal(t, "Old A", updated.Answer)
}

func TestCardService_Update_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestCardService()
	newQ := "Q"
	_, err := svc.Update(context.Background(), 999, 1, domain.UpdateCardRequest{Question: &newQ})
	require.ErrorIs(t, err, ErrCardNotFound)
}

func TestCardService_Update_Forbidden(t *testing.T) {
	svc, cards, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false)
	c := &domain.Card{DeckID: deck.ID, Question: "Q", Answer: "A"}
	_ = cards.Create(context.Background(), c)

	newQ := "Q2"
	_, err := svc.Update(context.Background(), c.ID, 2, domain.UpdateCardRequest{Question: &newQ})
	require.ErrorIs(t, err, ErrCardForbidden)
}

// ---- Delete ----------------------------------------------------------------

func TestCardService_Delete_Success(t *testing.T) {
	svc, cards, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false)
	c := &domain.Card{DeckID: deck.ID, Question: "Q", Answer: "A"}
	_ = cards.Create(context.Background(), c)

	err := svc.Delete(context.Background(), c.ID, 1)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), c.ID, 1)
	require.ErrorIs(t, err, ErrCardNotFound)
}

func TestCardService_Delete_NotFound(t *testing.T) {
	svc, _, _, _, _ := newTestCardService()
	err := svc.Delete(context.Background(), 999, 1)
	require.ErrorIs(t, err, ErrCardNotFound)
}

func TestCardService_Delete_Forbidden(t *testing.T) {
	svc, cards, decks, _, _ := newTestCardService()
	deck := seedDeck(t, decks, 1, false)
	c := &domain.Card{DeckID: deck.ID, Question: "Q", Answer: "A"}
	_ = cards.Create(context.Background(), c)

	err := svc.Delete(context.Background(), c.ID, 2)
	require.ErrorIs(t, err, ErrCardForbidden)
}

// ---- Tags assignment -------------------------------------------------------

func TestCardService_Create_WithTags(t *testing.T) {
	svc, _, decks, _, tags := newTestCardService()
	deck := seedDeck(t, decks, 1, false)
	tag := &domain.Tag{Name: "go"}
	require.NoError(t, tags.Create(context.Background(), tag))

	card, err := svc.Create(context.Background(), deck.ID, 1, domain.CreateCardRequest{
		Question: "Q",
		Answer:   "A",
		TagIDs:   []int{tag.ID},
	})
	require.NoError(t, err)
	require.Len(t, card.Tags, 1)
	require.Equal(t, "go", card.Tags[0].Name)
}
