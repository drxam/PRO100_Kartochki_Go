package service

import (
	"context"
	"testing"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
	"github.com/stretchr/testify/require"
)

func newTestCategoryService() (*CategoryService, *categoryStoreMock) {
	store := newCategoryStoreMock()
	svc := NewCategoryService(store)
	return svc, store
}

func TestCategoryService_Create_Success(t *testing.T) {
	svc, _ := newTestCategoryService()
	cat, err := svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "Математика"})
	require.NoError(t, err)
	require.NotNil(t, cat)
	require.Equal(t, "Математика", cat.Name)
	require.Equal(t, 1, cat.ID)
}

func TestCategoryService_Create_Duplicate(t *testing.T) {
	svc, _ := newTestCategoryService()
	_, err := svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "Физика"})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "Физика"})
	require.ErrorIs(t, err, ErrCategoryExists)
}

func TestCategoryService_List(t *testing.T) {
	svc, _ := newTestCategoryService()
	_, _ = svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "A"})
	_, _ = svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "B"})

	list, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
}

func TestCategoryService_List_Empty(t *testing.T) {
	svc, _ := newTestCategoryService()
	list, err := svc.List(context.Background())
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestCategoryService_GetByID_Success(t *testing.T) {
	svc, _ := newTestCategoryService()
	created, _ := svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "История"})

	found, err := svc.GetByID(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "История", found.Name)
}

func TestCategoryService_GetByID_NotFound(t *testing.T) {
	svc, _ := newTestCategoryService()
	_, err := svc.GetByID(context.Background(), 999)
	require.ErrorIs(t, err, ErrCategoryNotFound)
}

func TestCategoryService_Update_Success(t *testing.T) {
	svc, _ := newTestCategoryService()
	created, _ := svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "Старое"})

	updated, err := svc.Update(context.Background(), created.ID, domain.UpdateCategoryRequest{Name: "Новое"})
	require.NoError(t, err)
	require.Equal(t, "Новое", updated.Name)
}

func TestCategoryService_Update_NotFound(t *testing.T) {
	svc, _ := newTestCategoryService()
	_, err := svc.Update(context.Background(), 999, domain.UpdateCategoryRequest{Name: "Что-то"})
	require.ErrorIs(t, err, ErrCategoryNotFound)
}

func TestCategoryService_Update_DuplicateName(t *testing.T) {
	svc, _ := newTestCategoryService()
	_, _ = svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "Занято"})
	cat2, _ := svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "Другое"})

	_, err := svc.Update(context.Background(), cat2.ID, domain.UpdateCategoryRequest{Name: "Занято"})
	require.ErrorIs(t, err, ErrCategoryExists)
}

func TestCategoryService_Update_SameNameAllowed(t *testing.T) {
	svc, _ := newTestCategoryService()
	cat, _ := svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "Одно"})

	// Обновление с тем же именем для той же категории — допустимо
	updated, err := svc.Update(context.Background(), cat.ID, domain.UpdateCategoryRequest{Name: "Одно"})
	require.NoError(t, err)
	require.Equal(t, "Одно", updated.Name)
}

func TestCategoryService_Delete_Success(t *testing.T) {
	svc, _ := newTestCategoryService()
	cat, _ := svc.Create(context.Background(), domain.CreateCategoryRequest{Name: "Удали"})

	err := svc.Delete(context.Background(), cat.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(context.Background(), cat.ID)
	require.ErrorIs(t, err, ErrCategoryNotFound)
}

func TestCategoryService_Delete_NotFound(t *testing.T) {
	svc, _ := newTestCategoryService()
	err := svc.Delete(context.Background(), 999)
	require.ErrorIs(t, err, ErrCategoryNotFound)
}
