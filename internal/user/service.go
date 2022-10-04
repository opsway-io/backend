package user

import (
	"context"
	"fmt"
	"net/http"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/storage"
	"github.com/pkg/errors"
)

type Service interface {
	GetUserAndTeamsByUserID(ctx context.Context, userId uint) (*entities.User, error)
	GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error)
	Create(ctx context.Context, user *entities.User) error
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uint) error
	ScrapeUserAvatarFromURL(ctx context.Context, userID uint, URL string) error
	GetUserAvatarURL(ctx context.Context, user *entities.User) (URL *string)
}

type ServiceImpl struct {
	repository Repository
	storage    storage.Service
}

func NewService(repository Repository, storage storage.Service) Service {
	return &ServiceImpl{
		repository: repository,
		storage:    storage,
	}
}

func (s *ServiceImpl) GetUserAndTeamsByUserID(ctx context.Context, userId uint) (*entities.User, error) {
	return s.repository.GetUserAndTeamsByUserID(ctx, userId)
}

func (s *ServiceImpl) GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error) {
	return s.repository.GetUserAndTeamsByEmailAddress(ctx, email)
}

func (s *ServiceImpl) Create(ctx context.Context, user *entities.User) error {
	return s.repository.Create(ctx, user)
}

func (s *ServiceImpl) Update(ctx context.Context, user *entities.User) error {
	return s.repository.Update(ctx, user)
}

func (s *ServiceImpl) Delete(ctx context.Context, id uint) error {
	return s.repository.Delete(ctx, id)
}

func (s *ServiceImpl) ScrapeUserAvatarFromURL(ctx context.Context, userID uint, URL string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return errors.Wrap(err, "failed to get avatar from URL")
	}

	defer resp.Body.Close()

	key := s.getUserAvatarKey(userID)
	err = s.storage.PutFile(ctx, "avatars", key, resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to upload avatar to storage")
	}

	s.repository.Update(ctx, &entities.User{
		ID:     userID,
		Avatar: &key,
	})

	return nil
}

func (s *ServiceImpl) GetUserAvatarURL(ctx context.Context, user *entities.User) *string {
	if user.Avatar != nil {
		return nil
	}

	key := s.getUserAvatarKey(user.ID)
	url := s.storage.GetPublicFileURL("avatars", key)

	return &url
}

func (s *ServiceImpl) getUserAvatarKey(userID uint) string {
	return fmt.Sprintf("users/%d", userID)
}
