package service

import (
	"context"
	"testing"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/stretchr/testify/require"
)

func newTestTagService() (*TagService, *tagStoreMock) {
	store := newTagStoreMock()
	svc := NewTagService(store)
	return svc, store
}

func TestTagService_Create_Success(t *testing.T) {
	svc, _ := newTestTagService()
	tag, err := svc.Create(context.Background(), domain.CreateTagRequest{Name: "golang"})
	require.NoError(t, err)
	require.Equal(t, "golang", tag.Name)
	require.Equal(t, 1, tag.ID)
}

func TestTagService_Create_Duplicate(t *testing.T) {
	svc, _ := newTestTagService()
	_, err := svc.Create(context.Background(), domain.CreateTagRequest{Name: "python"})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), domain.CreateTagRequest{Name: "python"})
	require.ErrorIs(t, err, ErrTagExists)
}

func TestTagService_List(t *testing.T) {
	svc, _ := newTestTagService()
	_, _ = svc.Create(context.Background(), domain.CreateTagRequest{Name: "tag1"})
	_, _ = svc.Create(context.Background(), domain.CreateTagRequest{Name: "tag2"})

	list, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestTagService_List_Empty(t *testing.T) {
	svc, _ := newTestTagService()
	list, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestTagService_GetByID_Success(t *testing.T) {
	svc, _ := newTestTagService()
	created, _ := svc.Create(context.Background(), domain.CreateTagRequest{Name: "go"})

	found, err := svc.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "go", found.Name)
}

func TestTagService_GetByID_NotFound(t *testing.T) {
	svc, _ := newTestTagService()
	_, err := svc.GetByID(context.Background(), 999)
	require.ErrorIs(t, err, ErrTagNotFound)
}

func TestTagService_Update_Success(t *testing.T) {
	svc, _ := newTestTagService()
	tag, _ := svc.Create(context.Background(), domain.CreateTagRequest{Name: "old"})

	updated, err := svc.Update(context.Background(), tag.ID, domain.UpdateTagRequest{Name: "new"})
	require.NoError(t, err)
	require.Equal(t, "new", updated.Name)
}

func TestTagService_Update_NotFound(t *testing.T) {
	svc, _ := newTestTagService()
	_, err := svc.Update(context.Background(), 999, domain.UpdateTagRequest{Name: "x"})
	require.ErrorIs(t, err, ErrTagNotFound)
}

func TestTagService_Update_DuplicateName(t *testing.T) {
	svc, _ := newTestTagService()
	_, _ = svc.Create(context.Background(), domain.CreateTagRequest{Name: "taken"})
	tag2, _ := svc.Create(context.Background(), domain.CreateTagRequest{Name: "free"})

	_, err := svc.Update(context.Background(), tag2.ID, domain.UpdateTagRequest{Name: "taken"})
	require.ErrorIs(t, err, ErrTagExists)
}

func TestTagService_Update_SameNameAllowed(t *testing.T) {
	svc, _ := newTestTagService()
	tag, _ := svc.Create(context.Background(), domain.CreateTagRequest{Name: "same"})

	updated, err := svc.Update(context.Background(), tag.ID, domain.UpdateTagRequest{Name: "same"})
	require.NoError(t, err)
	require.Equal(t, "same", updated.Name)
}

func TestTagService_Delete_Success(t *testing.T) {
	svc, _ := newTestTagService()
	tag, _ := svc.Create(context.Background(), domain.CreateTagRequest{Name: "remove"})

	err := svc.Delete(context.Background(), tag.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), tag.ID)
	require.ErrorIs(t, err, ErrTagNotFound)
}

func TestTagService_Delete_NotFound(t *testing.T) {
	svc, _ := newTestTagService()
	err := svc.Delete(context.Background(), 999)
	require.ErrorIs(t, err, ErrTagNotFound)
}
