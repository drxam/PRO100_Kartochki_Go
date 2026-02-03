package service

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/pro100kartochki/mozgoemka/internal/domain"
	"github.com/pro100kartochki/mozgoemka/internal/repository"
)

type UserService struct {
	userRepo    *repository.UserRepository
	deckRepo    *repository.DeckRepository
	cardRepo    *repository.CardRepository
	uploadPath  string // корень загрузок (например ./uploads)
	baseURL     string // например http://localhost:8080
}

func NewUserService(userRepo *repository.UserRepository, deckRepo *repository.DeckRepository, cardRepo *repository.CardRepository) *UserService {
	return &UserService{userRepo: userRepo, deckRepo: deckRepo, cardRepo: cardRepo}
}

func (s *UserService) SetUploadConfig(uploadPath, baseURL string) {
	s.uploadPath = uploadPath
	s.baseURL = baseURL
}

func (s *UserService) GetByID(ctx context.Context, id int) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, id)
}

func (s *UserService) GetProfile(ctx context.Context, userID int) (*domain.UserProfileResponse, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, err
	}
	decksCount, _ := s.deckRepo.CountByUserID(ctx, userID)
	cardsCount, _ := s.cardRepo.CountByUserID(ctx, userID)
	return &domain.UserProfileResponse{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		AvatarURL: u.AvatarURL,
		Role:      u.Role,
		Stats:     domain.UserStats{DecksCount: decksCount, CardsCount: cardsCount},
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, userID int, req domain.UpdateProfileRequest) (*domain.User, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || u == nil {
		return nil, err
	}
	if req.Username != nil {
		u.Username = req.Username
	}
	if err := s.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}
	u.PasswordHash = ""
	return u, nil
}

// UploadAvatar сохраняет файл аватара в uploadPath/avatars/{uuid}.ext и возвращает URL.
func (s *UserService) UploadAvatar(ctx context.Context, userID int, filename string, data []byte) (avatarURL string, err error) {
	ext := ".jpg"
	if len(filename) > 4 {
		ext = filename[len(filename)-4:]
		if ext != ".jpg" && ext != "jpeg" && ext != ".png" && ext != ".PNG" && ext != ".JPG" {
			ext = ".jpg"
		}
	}
	name := uuid.New().String() + ext
	avatarPath := filepath.Join(s.uploadPath, "avatars")
	_ = os.MkdirAll(avatarPath, 0755)
	fullPath := filepath.Join(avatarPath, name)
	if err = os.WriteFile(fullPath, data, 0644); err != nil {
		return "", err
	}
	url := s.baseURL + "/uploads/avatars/" + name
	u, _ := s.userRepo.GetByID(ctx, userID)
	if u != nil {
		u.AvatarURL = &url
		_ = s.userRepo.Update(ctx, u)
	}
	return url, nil
}
