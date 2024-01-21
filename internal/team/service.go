package team

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/notification/email"
	"github.com/opsway-io/backend/internal/notification/email/templates"
	"github.com/opsway-io/backend/internal/storage"
	"github.com/pkg/errors"
)

var (
	ErrAlreadyOnTeam          = errors.New("user is already on team")
	ErrInvalidInvitationToken = errors.New("invalid invitation token")
)

type Config struct {
	AvatarBucket     string        `mapstructure:"avatars" default:"avatars"`
	InvitationExpiry time.Duration `mapstructure:"invitation_expiry" default:"168h"` // 7 days default
	InvitationSecret string        `mapstructure:"invitation_secret" required:"true"`
	ApplicationURL   string        `mapstructure:"application_url" required:"true"`
}

type Service interface {
	CreateWithOwnerUserID(ctx context.Context, team *entities.Team, ownerUserID uint) error

	GetByID(ctx context.Context, teamId uint) (*entities.Team, error)

	GetByStripeID(ctx context.Context, stripeID string) (*entities.Team, error)

	GetTeamsAndRoleByUserID(ctx context.Context, userID uint) (*[]TeamAndRole, error)
	GetUsersByID(ctx context.Context, teamId uint, offset *int, limit *int, query *string) (*[]TeamUser, error)
	GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error)
	UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.TeamRole) error

	UpdateBilling(ctx context.Context, teamID uint, customerID string, plan string) error

	RemoveUser(ctx context.Context, teamID, userID uint) error
	UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error

	Delete(ctx context.Context, id uint) error
	UploadAvatar(ctx context.Context, teamID uint, file io.Reader) error
	DeleteAvatar(ctx context.Context, teamID uint) error
	GetAvatarURLByID(teamID uint) (URL string)

	IsNameAvailable(ctx context.Context, name string) (bool, error)

	InviteByEmail(ctx context.Context, teamID uint, role entities.TeamRole, email string) error
	GenerateInviteLink(ctx context.Context, teamID uint, role entities.TeamRole, email string) (string, error)
	AcceptInviteByToken(ctx context.Context, token string, user *entities.User) error
}

type ServiceImpl struct {
	config     Config
	repository Repository
	storage    storage.Service
	email      email.Sender
}

func NewService(cfg Config, repository Repository, storage storage.Service, email email.Sender) Service {
	return &ServiceImpl{
		config:     cfg,
		repository: repository,
		storage:    storage,
		email:      email,
	}
}

func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.Team, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) GetByStripeID(ctx context.Context, stripeID string) (*entities.Team, error) {
	return s.repository.GetByStripeID(ctx, stripeID)
}

func (s *ServiceImpl) CreateWithOwnerUserID(ctx context.Context, team *entities.Team, ownerUserID uint) error {
	return s.repository.CreateWithOwnerUserID(ctx, team, ownerUserID)
}

func (s *ServiceImpl) UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error {
	return s.repository.UpdateDisplayName(ctx, teamID, displayName)
}

func (s *ServiceImpl) Delete(ctx context.Context, id uint) error {
	if err := s.DeleteAvatar(ctx, id); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return errors.Wrap(err, "failed to delete avatar")
		}
	}

	return s.repository.Delete(ctx, id)
}

func (s *ServiceImpl) GetUsersByID(ctx context.Context, id uint, offset *int, limit *int, query *string) (*[]TeamUser, error) {
	return s.repository.GetUsersByID(ctx, id, offset, limit, query)
}

func (s *ServiceImpl) GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error) {
	return s.repository.GetUserRole(ctx, teamID, userID)
}

func (s *ServiceImpl) RemoveUser(ctx context.Context, teamID, userID uint) error {
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
	return s.repository.UpdateUserRole(ctx, teamID, userID, role)
}

func (s *ServiceImpl) UpdateBilling(ctx context.Context, teamID uint, customerID string, plan string) error {
	return s.repository.UpdateBilling(ctx, teamID, customerID, plan)
}

func (s *ServiceImpl) GetTeamsAndRoleByUserID(ctx context.Context, userID uint) (*[]TeamAndRole, error) {
	return s.repository.GetTeamsAndRoleByUserID(ctx, userID)
}

func (s *ServiceImpl) IsNameAvailable(ctx context.Context, name string) (bool, error) {
	return s.repository.IsNameAvailable(ctx, name)
}

func (s *ServiceImpl) InviteByEmail(ctx context.Context, teamID uint, role entities.TeamRole, email string) error {
	// Check if user is already on the team
	isOnTeam, err := s.repository.IsUserOnTeamByEmail(ctx, teamID, email)
	if err != nil {
		return errors.Wrap(err, "failed to check if user is on team")
	}
	if isOnTeam {
		return ErrAlreadyOnTeam
	}

	// Get team name
	team, err := s.repository.GetByID(ctx, teamID)
	if err != nil {
		return errors.Wrap(err, "failed to get team")
	}

	// Generate invite link
	link, err := s.GenerateInviteLink(ctx, teamID, role, email)
	if err != nil {
		return errors.Wrap(err, "failed to generate invite link")
	}

	// Send email
	if err := s.email.Send(
		ctx,
		"",
		email,
		&templates.TeamInvitationTemplate{
			TeamName:       team.Name,
			ActivationLink: link,
		},
	); err != nil {
		return errors.Wrap(err, "failed to send email")
	}

	return nil
}

func (s *ServiceImpl) GenerateInviteLink(ctx context.Context, teamID uint, role entities.TeamRole, email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp":     time.Now().Add(s.config.InvitationExpiry).Unix(),
		"type":    "team-invite",
		"sub":     email,
		"team_id": teamID,
		"role":    role,
	})

	tokenString, err := token.SignedString([]byte(s.config.InvitationSecret))
	if err != nil {
		return "", errors.Wrap(err, "failed to sign token")
	}

	return fmt.Sprintf("%s/login/team/invite?token=%s", s.config.ApplicationURL, tokenString), nil
}

func (s *ServiceImpl) AcceptInviteByToken(ctx context.Context, tokenString string, user *entities.User) error {
	// Parse and validate token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(s.config.InvitationSecret), nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to parse token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("failed to parse token claims")
	}
	if !token.Valid {
		return errors.New("invalid token")
	}

	// Check token type
	if claims["type"] != "team-invite" {
		return errors.New("invalid token type")
	}

	// Check user email matches token email
	if claims["sub"] != user.Email {
		return errors.New("token email does not match user email")
	}

	// Get role
	role, err := entities.TeamRoleFrom(claims["role"])
	if err != nil {
		return errors.Wrap(err, "failed to get role")
	}

	fmt.Println(claims)

	// Get team ID
	teamID := uint(claims["team_id"].(float64))

	// Add user to team
	if err := s.repository.AddUser(
		ctx,
		uint(teamID),
		user.ID,
		role,
	); err != nil {
		return errors.Wrap(err, "failed to add user to team")
	}

	return nil
}

func (s *ServiceImpl) getAvatarKey(teamID uint) string {
	return fmt.Sprintf("teams/%d", teamID)
}
