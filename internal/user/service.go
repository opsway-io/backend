package user

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/email/templates"
	"github.com/opsway-io/backend/internal/storage"
	"github.com/pkg/errors"
)

var ErrInvalidPassword = errors.New("invalid password")

type Service interface {
	Create(ctx context.Context, user *entities.User) error
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uint) error

	GetUserAndTeamsByUserID(ctx context.Context, userId uint) (*entities.User, error)
	GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error)

	SetAvatarFromURL(ctx context.Context, userID uint, URL string) error
	GetAvatarURLByID(userID uint) (URL string)
	UploadAvatar(ctx context.Context, userID uint, file io.Reader) error
	DeleteAvatar(ctx context.Context, userID uint) error

	ChangePasswordWithOldPassword(ctx context.Context, userID uint, oldPassword string, newPassword string) error
	ChangePasswordWithResetToken(ctx context.Context, userID uint, token string, newPassword string) (err error)
	RequestPasswordReset(ctx context.Context, userId uint) error
}

type ServiceImpl struct {
	repository Repository
	storage    storage.Service
	cache      Cache
	email      email.Sender
}

func NewService(repository Repository, cache Cache, storage storage.Service, email email.Sender) Service {
	return &ServiceImpl{
		repository: repository,
		cache:      cache,
		storage:    storage,
		email:      email,
	}
}

func (s *ServiceImpl) GetUserAndTeamsByUserID(ctx context.Context, userId uint) (*entities.User, error) {
	return s.repository.GetUserAndTeamsByUserID(ctx, userId)
}

func (s *ServiceImpl) GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error) {
	return s.repository.GetUserAndTeamsByEmailAddress(ctx, email)
}

func (s *ServiceImpl) Create(ctx context.Context, user *entities.User) error {
	err := s.repository.Create(ctx, user)
	if err != nil {
		return errors.Wrap(err, "failed to create user")
	}

	if err := s.email.Send(
		ctx,
		user.Name,
		user.Email,
		&templates.NewUserWelcomeTemplate{
			Name: user.Name,
		},
	); err != nil {
		return errors.Wrap(err, "failed to send welcome email")
	}

	return nil
}

func (s *ServiceImpl) Update(ctx context.Context, user *entities.User) error {
	return s.repository.Update(ctx, user)
}

func (s *ServiceImpl) Delete(ctx context.Context, id uint) error {
	return s.repository.Delete(ctx, id)
}

func (s *ServiceImpl) SetAvatarFromURL(ctx context.Context, userID uint, URL string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return errors.Wrap(err, "failed to get avatar from URL")
	}

	defer resp.Body.Close()

	key := s.getAvatarKey(userID)
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

func (s *ServiceImpl) GetAvatarURLByID(userID uint) string {
	key := s.getAvatarKey(userID)

	return s.storage.GetPublicFileURL("avatars", key)
}

func (s *ServiceImpl) getAvatarKey(userID uint) string {
	return fmt.Sprintf("users/%d", userID)
}

func (s *ServiceImpl) UploadAvatar(ctx context.Context, userID uint, file io.Reader) error {
	key := s.getAvatarKey(userID)

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

func (s *ServiceImpl) DeleteAvatar(ctx context.Context, userID uint) error {
	if err := s.repository.Update(ctx, &entities.User{
		ID:        userID,
		HasAvatar: false,
	}); err != nil {
		return errors.Wrap(err, "failed to update user")
	}

	key := s.getAvatarKey(userID)

	err := s.storage.DeleteFile(ctx, "avatars", key)
	if err != nil {
		return errors.Wrap(err, "failed to delete avatar from storage")
	}

	return nil
}

func (s *ServiceImpl) ChangePasswordWithOldPassword(ctx context.Context, userID uint, oldPassword string, newPassword string) error {
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

func (s *ServiceImpl) RequestPasswordReset(ctx context.Context, userId uint) error {
	user, err := s.repository.GetUserByID(ctx, userId)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}

		return errors.Wrap(err, "failed to get user")
	}

	token, err := uuid.NewV4()
	if err != nil {
		return errors.Wrap(err, "failed to generate token")
	}

	if err = s.cache.SetPasswordResetToken(
		ctx,
		user.ID,
		token.String(),
		time.Duration(24)*time.Hour, // TODO: move to config
	); err != nil {
		return errors.Wrap(err, "failed to set token")
	}

	if err := s.email.Send(
		ctx,
		user.Name,
		user.Email,
		&templates.PasswordResetTemplate{
			Name: user.Name,
			PasswordResetLink: fmt.Sprintf(
				"%s/reset-password?token=%s",
				"https://my.opsway.io", // TODO: move to config
				token.String(),
			),
		},
	); err != nil {
		return errors.Wrap(err, "failed to send password reset email")
	}

	return nil
}

func (s *ServiceImpl) ChangePasswordWithResetToken(ctx context.Context, userID uint, token string, newPassword string) error {
	tokenUserID, err := s.cache.VerifyAndDeletePasswordResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}

		return errors.Wrap(err, "failed to get user ID by token")

	}

	if userID != tokenUserID {
		return ErrNotFound
	}

	user, err := s.repository.GetUserByID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}

		return errors.Wrap(err, "failed to get user")
	}

	if err := user.SetPassword(newPassword); err != nil {
		return errors.Wrap(err, "failed to set new password")
	}

	if err := s.repository.Update(ctx, user); err != nil {
		return errors.Wrap(err, "failed to update user")
	}

	return nil
}
