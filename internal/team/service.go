package team

import (
	"context"
	"fmt"
	"io"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/storage"
	"github.com/pkg/errors"
)

type Service interface {
	GetByID(ctx context.Context, teamId uint) (*entities.Team, error)
	GetUsersByID(ctx context.Context, teamId uint, offset *int, limit *int, query *string) (*[]TeamUser, error)
	GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error)
	RemoveUser(ctx context.Context, teamID, userID uint) error
	Create(ctx context.Context, team *entities.Team) error
	UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error
	UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.TeamRole) error
	Delete(ctx context.Context, id uint) error
	UploadAvatar(ctx context.Context, teamID uint, file io.Reader) error
	DeleteAvatar(ctx context.Context, teamID uint) error
	GetAvatarURLByID(teamID uint) (URL string)
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

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.Team, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) Create(ctx context.Context, team *entities.Team) error {
	return s.repository.Create(ctx, team)
}

func (s *ServiceImpl) UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error {
	return s.repository.UpdateDisplayName(ctx, teamID, displayName)
}

func (s *ServiceImpl) Delete(ctx context.Context, id uint) error {
	return s.repository.Delete(ctx, id)
}

func (s *ServiceImpl) GetUsersByID(ctx context.Context, id uint, offset *int, limit *int, query *string) (*[]TeamUser, error) {
	return s.repository.GetUsersByID(ctx, id, offset, limit, query)
}

func (s *ServiceImpl) GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error) {
	return s.repository.GetUserRole(ctx, teamID, userID)
}

func (s *ServiceImpl) RemoveUser(ctx context.Context, teamID, userID uint) error {
	// TODO: make sure team still has at least one owner

	return s.repository.RemoveUser(ctx, teamID, userID)
}

func (s *ServiceImpl) UploadAvatar(ctx context.Context, teamID uint, file io.Reader) error {
	key := s.getAvatarKey(teamID)

	err := s.storage.PutFile(ctx, "avatars", key, file)
	if err != nil {
		return errors.Wrap(err, "failed to upload avatar to storage")
	}

	if err := s.repository.Update(ctx, &entities.Team{
		ID:        teamID,
		HasAvatar: true,
	}); err != nil {
		return errors.Wrap(err, "failed to update team")
	}

	return nil
}

func (s *ServiceImpl) DeleteAvatar(ctx context.Context, teamID uint) error {
	if err := s.repository.Update(ctx, &entities.Team{
		ID:        teamID,
		HasAvatar: false,
	}); err != nil {
		return errors.Wrap(err, "failed to update team")
	}

	key := s.getAvatarKey(teamID)

	err := s.storage.DeleteFile(ctx, "avatars", key)
	if err != nil {
		return errors.Wrap(err, "failed to delete avatar from storage")
	}

	return nil
}

func (s *ServiceImpl) GetAvatarURLByID(teamID uint) string {
	key := s.getAvatarKey(teamID)

	return s.storage.GetPublicFileURL("avatars", key)
}

func (s *ServiceImpl) UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.TeamRole) error {
	// TODO: make sure team still has at least one owner

	return s.repository.UpdateUserRole(ctx, teamID, userID, role)
}

func (s *ServiceImpl) getAvatarKey(teamID uint) string {
	return fmt.Sprintf("teams/%d", teamID)
}
