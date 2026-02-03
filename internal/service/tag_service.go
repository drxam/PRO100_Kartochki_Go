package service

import (
	"context"
	"errors"

	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/repository"
)

var ErrTagExists = errors.New("тег с таким именем уже существует")

type TagService struct {
	repo *repository.TagRepository
}

func NewTagService(repo *repository.TagRepository) *TagService {
	return &TagService{repo: repo}
}

func (s *TagService) Create(ctx context.Context, req domain.CreateTagRequest) (*domain.Tag, error) {
	existing, _ := s.repo.GetByName(ctx, req.Name)
	if existing != nil {
		return nil, ErrTagExists
	}
	t := &domain.Tag{Name: req.Name}
	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TagService) List(ctx context.Context) ([]domain.Tag, error) {
	return s.repo.List(ctx)
}

func (s *TagService) ListWithSearch(ctx context.Context, search string) ([]domain.Tag, error) {
	return s.repo.ListWithSearch(ctx, search)
}

func (s *TagService) GetByID(ctx context.Context, id int) (*domain.Tag, error) {
	return s.repo.GetByID(ctx, id)
}
