package service

import (
	"context"
	"errors"

	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/repository"
)

var ErrCategoryExists = errors.New("категория с таким именем уже существует")

type CategoryService struct {
	repo *repository.CategoryRepository
}

func NewCategoryService(repo *repository.CategoryRepository) *CategoryService {
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
	return s.repo.GetByID(ctx, id)
}
