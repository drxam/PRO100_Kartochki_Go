package service

import (
	"context"
	"errors"

	"github.com/drxam/PRO100_Kartochki_Go/internal/domain"
<<<<<<< Updated upstream
	"github.com/drxam/PRO100_Kartochki_Go/internal/repository"
=======
>>>>>>> Stashed changes
)

var (
	ErrTagExists   = errors.New("тег с таким именем уже существует")
	ErrTagNotFound = errors.New("тег не найден")
)

// TagStore — контракт репозитория тегов для unit-тестов.
type TagStore interface {
	Create(ctx context.Context, t *domain.Tag) error
	GetByID(ctx context.Context, id int) (*domain.Tag, error)
	GetByName(ctx context.Context, name string) (*domain.Tag, error)
	GetByIDs(ctx context.Context, ids []int) ([]domain.Tag, error)
	List(ctx context.Context) ([]domain.Tag, error)
	ListWithSearch(ctx context.Context, search string) ([]domain.Tag, error)
	Update(ctx context.Context, t *domain.Tag) error
	Delete(ctx context.Context, id int) error
}

type TagService struct {
	repo TagStore
}

func NewTagService(repo TagStore) *TagService {
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
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, ErrTagNotFound
	}
	return t, nil
}

func (s *TagService) Update(ctx context.Context, id int, req domain.UpdateTagRequest) (*domain.Tag, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil || t == nil {
		return nil, ErrTagNotFound
	}
	existing, _ := s.repo.GetByName(ctx, req.Name)
	if existing != nil && existing.ID != id {
		return nil, ErrTagExists
	}
	t.Name = req.Name
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *TagService) Delete(ctx context.Context, id int) error {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil || t == nil {
		return ErrTagNotFound
	}
	return s.repo.Delete(ctx, id)
}
