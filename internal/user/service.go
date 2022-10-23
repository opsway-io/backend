package user

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/storage"
	"github.com/pkg/errors"
)

var ErrInvalidPassword = errors.New("invalid password")

type Service interface {
	GetUserAndTeamsByUserID(ctx context.Context, userId uint) (*entities.User, error)
	GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error)
	Create(ctx context.Context, user *entities.User) error
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uint) error
	ScrapeUserAvatarFromURL(ctx context.Context, userID uint, URL string) error
	GetUserAvatarURLByID(userID uint) (URL string)
	DeleteUserAvatar(ctx context.Context, userID uint) error
	UploadUserAvatar(ctx context.Context, userID uint, file io.Reader) error
	ChangePassword(ctx context.Context, userID uint, oldPassword string, newPassword string) error
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
		ID:        userID,
		HasAvatar: true,
	})

	return nil
}

func (s *ServiceImpl) GetUserAvatarURLByID(userID uint) string {
	key := s.getUserAvatarKey(userID)

	return s.storage.GetPublicFileURL("avatars", key)
}

func (s *ServiceImpl) getUserAvatarKey(userID uint) string {
	return fmt.Sprintf("users/%d", userID)
}

func (s *ServiceImpl) DeleteUserAvatar(ctx context.Context, userID uint) error {
	if err := s.repository.Update(ctx, &entities.User{
		ID:        userID,
		HasAvatar: false,
	}); err != nil {
		return errors.Wrap(err, "failed to update user")
	}

	key := s.getUserAvatarKey(userID)

	err := s.storage.DeleteFile(ctx, "avatars", key)
	if err != nil {
		return errors.Wrap(err, "failed to delete avatar from storage")
	}

	return nil
}

func (s *ServiceImpl) UploadUserAvatar(ctx context.Context, userID uint, file io.Reader) error {
	key := s.getUserAvatarKey(userID)

	err := s.storage.PutFile(ctx, "avatars", key, file)
	if err != nil {
		return errors.Wrap(err, "failed to upload avatar to storage")
	}

	if err := s.repository.Update(ctx, &entities.User{
		ID:        userID,
		HasAvatar: true,
	}); err != nil {
		return errors.Wrap(err, "failed to update user")
	}

	return nil
}

func (s *ServiceImpl) ChangePassword(ctx context.Context, userID uint, oldPassword string, newPassword string) error {
	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}

		return errors.Wrap(err, "failed to get user")
	}

	if !user.CheckPassword(oldPassword) {
		return ErrInvalidPassword
	}

	if err := user.SetPassword(newPassword); err != nil {
		return errors.Wrap(err, "failed to set new password")
	}

	if err := s.repository.Update(ctx, user); err != nil {
		return errors.Wrap(err, "failed to update user")
	}

	return nil
}
