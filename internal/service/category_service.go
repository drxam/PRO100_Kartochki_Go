package service

import (
	"context"
	"errors"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
)

var (
	ErrCategoryExists   = errors.New("категория с таким именем уже существует")
	ErrCategoryNotFound = errors.New("категория не найдена")
)

// CategoryStore — контракт репозитория категорий для unit-тестов.
type CategoryStore interface {
	Create(ctx context.Context, c *domain.Category) error
	GetByID(ctx context.Context, id int) (*domain.Category, error)
	GetByName(ctx context.Context, name string) (*domain.Category, error)
	List(ctx context.Context) ([]domain.Category, error)
	Update(ctx context.Context, c *domain.Category) error
	Delete(ctx context.Context, id int) error
}

type CategoryService struct {
	repo CategoryStore
}

func NewCategoryService(repo CategoryStore) *CategoryService {
	return &CategoryService{repo: repo}
}

func (s *CategoryService) Create(ctx context.Context, req domain.CreateCategoryRequest) (*domain.Category, error) {
	existing, _ := s.repo.GetByName(ctx, req.Name)
	if existing != nil {
		return nil, ErrCategoryExists
	}
	c := &domain.Category{Name: req.Name}
	if err := s.repo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) List(ctx context.Context) ([]domain.Category, error) {
	return s.repo.List(ctx)
}

func (s *CategoryService) GetByID(ctx context.Context, id int) (*domain.Category, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, ErrCategoryNotFound
	}
	return c, nil
}

func (s *CategoryService) Update(ctx context.Context, id int, req domain.UpdateCategoryRequest) (*domain.Category, error) {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil || c == nil {
		return nil, ErrCategoryNotFound
	}
	existing, _ := s.repo.GetByName(ctx, req.Name)
	if existing != nil && existing.ID != id {
		return nil, ErrCategoryExists
	}
	c.Name = req.Name
	if err := s.repo.Update(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CategoryService) Delete(ctx context.Context, id int) error {
	c, err := s.repo.GetByID(ctx, id)
	if err != nil || c == nil {
		return ErrCategoryNotFound
	}
	return s.repo.Delete(ctx, id)
}
