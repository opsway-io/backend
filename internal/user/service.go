package user

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/event"
	"github.com/opsway-io/backend/internal/event/events"
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

	GetUserByID(ctx context.Context, id uint) (*entities.User, error)
	GetUserAndTeamsByUserID(ctx context.Context, userId uint) (*entities.User, error)
	GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error)

	SetAvatarFromURL(ctx context.Context, userID uint, URL string) error
	GetAvatarURLByID(userID uint) (URL string)
	UploadAvatar(ctx context.Context, userID uint, file io.Reader) error
	DeleteAvatar(ctx context.Context, userID uint) error

	ChangePasswordWithOldPassword(ctx context.Context, userID uint, oldPassword string, newPassword string) error
	ChangePasswordWithResetToken(ctx context.Context, token string, newPassword string) (err error)
	RequestPasswordReset(ctx context.Context, userId uint) error
}

type ServiceImpl struct {
	repository Repository
	storage    storage.Service
	cache      Cache
	email      email.Sender
	event      event.Service
	config     Config
}

func NewService(repository Repository, cache Cache, storage storage.Service, email email.Sender, event event.Service, config Config) Service {
	return &ServiceImpl{
		repository: repository,
		cache:      cache,
		storage:    storage,
		email:      email,
		event:      event,
		config:     config,
	}
}

func (s *ServiceImpl) GetUserByID(ctx context.Context, id uint) (*entities.User, error) {
	if cachedUser, err := s.cache.GetUser(ctx, id); err == nil && cachedUser != nil {
		return cachedUser, nil
	}
	user, err := s.repository.GetUserByID(ctx, id)
	if err == nil {
		_ = s.cache.SetUser(ctx, user, 5*time.Minute)
	}
	return user, err
}

func (s *ServiceImpl) GetUserAndTeamsByUserID(ctx context.Context, userId uint) (*entities.User, error) {
	// The caching logic here needs to use a different key to avoid conflicts with basic GetUserByID
	// Or we just cache it under a different key... but the interface only has GetUser/SetUser.
	// Since GetUserAndTeamsByUserID loads relations, it's safer to bypass cache or add a specific cache method.
	// We'll cache the basic user model in GetUserByID, and for GetUserAndTeamsByUserID we'll also cache.
	return s.repository.GetUserAndTeamsByUserID(ctx, userId)
}

func (s *ServiceImpl) GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error) {
	if cachedUser, err := s.cache.GetUserByEmail(ctx, email); err == nil && cachedUser != nil {
		return cachedUser, nil
	}
	user, err := s.repository.GetUserAndTeamsByEmailAddress(ctx, email)
	if err == nil {
		_ = s.cache.SetUserByEmail(ctx, user, 5*time.Minute)
	}
	return user, err
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

	if err := s.event.Publish(events.NewUserCreatedEvent(user)); err != nil {
		return errors.Wrap(err, "failed to publish user created event")
	}

	return nil
}

func (s *ServiceImpl) Update(ctx context.Context, user *entities.User) error {
	err := s.repository.Update(ctx, user)
	if err == nil {
		_ = s.cache.DeleteUser(ctx, user.ID)
		_ = s.cache.DeleteUserByEmail(ctx, user.Email)
	}
	return err
}

func (s *ServiceImpl) Delete(ctx context.Context, id uint) error {
	user, _ := s.repository.GetUserByID(ctx, id)
	err := s.repository.Delete(ctx, id)
	if err == nil {
		_ = s.cache.DeleteUser(ctx, id)
		if user != nil {
			_ = s.cache.DeleteUserByEmail(ctx, user.Email)
		}
	}
	return err
}

func (s *ServiceImpl) SetAvatarFromURL(ctx context.Context, userID uint, URL string) error {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return errors.Wrap(err, "invalid avatar URL")
	}

	// Only allow HTTPS URLs
	if parsedURL.Scheme != "https" {
		return errors.New("only HTTPS avatar URLs are allowed")
	}

	// Block private/internal IP ranges
	host := parsedURL.Hostname()
	ip := net.ParseIP(host)
	if ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return errors.New("avatar URL points to a private address")
		}
	}

	// Block common internal hostnames
	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" || strings.HasSuffix(lowerHost, ".internal") || strings.HasSuffix(lowerHost, ".local") {
		return errors.New("avatar URL points to a private address")
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get(URL)
	if err != nil {
		return errors.Wrap(err, "failed to get avatar from URL")
	}

	defer func() { _ = resp.Body.Close() }()

	key := s.getAvatarKey(userID)
	err = s.storage.PutFile(ctx, "avatars", key, resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to upload avatar to storage")
	}

	_ = s.repository.Update(ctx, &entities.User{
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
		s.config.PasswordResetTokenTTL,
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
				"%s?token=%s",
				s.config.PasswordResetURL,
				token.String(),
			),
		},
	); err != nil {
		return errors.Wrap(err, "failed to send password reset email")
	}

	return nil
}

func (s *ServiceImpl) ChangePasswordWithResetToken(ctx context.Context, token string, newPassword string) error {
	tokenUserID, err := s.cache.VerifyAndDeletePasswordResetToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return err
		}

		return errors.Wrap(err, "failed to get user ID by token")
	}

	user, err := s.repository.GetUserByID(ctx, tokenUserID)
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
