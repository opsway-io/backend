package team

import (
	"context"
	"errors"
	"strings"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound          = errors.New("team not found")
	ErrUserNotFound      = errors.New("team user not found")
	ErrNameAlreadyExists = errors.New("team name already exists")
	ErrCannotRemoveOwner = errors.New("cannot remove owner")
	ErrInviteNotFound    = errors.New("team invite not found")
)

type Repository interface {
	GetByID(ctx context.Context, teamId uint) (*entities.Team, error)
	GetByStripeID(ctx context.Context, stripeID string) (*entities.Team, error)
	GetUsersByID(ctx context.Context, teamId uint, offset *int, limit *int, query *string) (*[]TeamUser, error)
	GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error)
	GetTeamsAndRoleByUserID(ctx context.Context, userID uint) (*[]TeamAndRole, error)

	Update(ctx context.Context, team *entities.Team) error
	UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.TeamRole) error
	UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error

	UpdateBilling(ctx context.Context, teamID uint, customerID string, plan string) error
	UpdateTeam(ctx context.Context, team *entities.Team) error

	CreateWithOwnerUserID(ctx context.Context, team *entities.Team, ownerUserID uint) error

	Delete(ctx context.Context, id uint) error

	AddUser(ctx context.Context, teamID, userID uint, role entities.TeamRole) error
	RemoveUser(ctx context.Context, teamID, userID uint) error

	IsNameAvailable(ctx context.Context, name string) (bool, error)
	IsUserOnTeamByEmail(ctx context.Context, teamID uint, email string) (bool, error)

	UpdateSettings(ctx context.Context, team *entities.Team) error

	UpsertInvite(ctx context.Context, invite *entities.TeamInvite) error
	GetInvites(ctx context.Context, teamID uint) (*[]entities.TeamInvite, error)
	GetInvite(ctx context.Context, teamID uint, email string) (*entities.TeamInvite, error)
	DeleteInvite(ctx context.Context, teamID uint, email string) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (s *RepositoryImpl) GetByID(ctx context.Context, id uint) (*entities.Team, error) {
	var team entities.Team
	if err := s.db.WithContext(ctx).First(&team, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &team, nil
}

func (s *RepositoryImpl) GetByStripeID(ctx context.Context, stripeID string) (*entities.Team, error) {
	var team entities.Team
	if err := s.db.WithContext(ctx).Where("stripe_customer_id = ?", stripeID).First(&team).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &team, nil
}

type TeamUser struct {
	entities.User
	Role       entities.TeamRole
	TotalCount int
}

func (s *RepositoryImpl) GetUsersByID(ctx context.Context, teamId uint, offset *int, limit *int, query *string) (*[]TeamUser, error) {
	var users []TeamUser

	s.db.WithContext(ctx).
		Select("u.*, tu.role").
		Table("team_users as tu").
		Joins("INNER JOIN users as u ON u.id = tu.user_id AND tu.team_id = ?", teamId).
		Order("u.name ASC").
		Scopes(
			postgres.Paginated(offset, limit),
			postgres.IncludeTotalCount("total_count"),
			postgres.Search([]string{"u.name", "u.display_name", "u.email"}, query),
		).
		Find(&users)

	return &users, nil
}

func (s *RepositoryImpl) CreateWithOwnerUserID(ctx context.Context, team *entities.Team, ownerUserID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}

		teamUser := entities.TeamUser{
			TeamID: team.ID,
			UserID: ownerUserID,
			Role:   entities.TeamRoleOwner,
		}

		if err := tx.Create(&teamUser).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *RepositoryImpl) UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error {
	result := s.db.WithContext(ctx).Model(&entities.Team{}).Where(entities.Team{
		ID: teamID,
	}).Update("display_name", displayName)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) UpdateBilling(ctx context.Context, teamID uint, customerID string, plan string) error {
	result := s.db.WithContext(ctx).Model(&entities.Team{}).Where(entities.Team{
		ID: teamID,
	}).Updates(entities.Team{
		StripeCustomerID: &customerID,
		PaymentPlan:      entities.PaymentPlan(plan),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) UpdateTeam(ctx context.Context, team *entities.Team) error {
	result := s.db.WithContext(ctx).Save(team)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.TeamRole) error {
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// If role is owner, remove all other owners
		if role == entities.TeamRoleOwner {
			if err := tx.Model(&entities.TeamUser{}).Where(entities.TeamUser{
				TeamID: teamID,
				Role:   entities.TeamRoleOwner,
			}).Update("role", entities.TeamRoleAdmin).Error; err != nil {
				return err
			}
		}

		result := s.db.WithContext(ctx).Model(&entities.TeamUser{}).Where(entities.TeamUser{
			TeamID: teamID,
			UserID: userID,
		}).Update("role", role)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrUserNotFound
		}

		return nil
	})

	return err
}

func (s *RepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Select(clause.Associations).Delete(&entities.Team{
		ID: id,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error) {
	var teamUser entities.TeamUser
	if err := s.db.WithContext(ctx).Where(
		entities.TeamUser{
			TeamID: teamID,
			UserID: userID,
		},
	).First(&teamUser).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &teamUser.Role, nil
}

func (s *RepositoryImpl) AddUser(ctx context.Context, teamID, userID uint, role entities.TeamRole) error {
	return s.db.WithContext(ctx).
		Where(entities.TeamUser{
			TeamID: teamID,
			UserID: userID,
		}).
		Assign(entities.TeamUser{
			Role: role,
		}).
		FirstOrCreate(&entities.TeamUser{}).
		Error
}

func (s *RepositoryImpl) RemoveUser(ctx context.Context, teamID, userID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var teamUser entities.TeamUser
		if err := tx.Where(
			entities.TeamUser{
				TeamID: teamID,
				UserID: userID,
			},
		).First(&teamUser).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrUserNotFound
			}

			return err
		}

		if teamUser.Role == entities.TeamRoleOwner {
			return ErrCannotRemoveOwner
		}

		if err := tx.Delete(&teamUser).Error; err != nil {
			return err
		}

		return nil
	})
}

func (s *RepositoryImpl) Update(ctx context.Context, team *entities.Team) error {
	result := s.db.WithContext(ctx).Model(team).Updates(team)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

type TeamAndRole struct {
	entities.Team
	Role entities.TeamRole
}

func (s *RepositoryImpl) GetTeamsAndRoleByUserID(ctx context.Context, userID uint) (*[]TeamAndRole, error) {
	var teams []TeamAndRole

	s.db.WithContext(ctx).
		Select("t.*, tu.role").
		Table("team_users as tu").
		Joins("INNER JOIN teams as t ON t.id = tu.team_id").
		Where("tu.user_id = ?", userID).
		Find(&teams)

	return &teams, nil
}

func (s *RepositoryImpl) IsNameAvailable(ctx context.Context, name string) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&entities.Team{}).Where("name = ?", name).Count(&count).Error; err != nil {
		return false, err
	}

	return count == 0, nil
}

func (s *RepositoryImpl) IsUserOnTeamByEmail(ctx context.Context, teamID uint, email string) (bool, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&entities.TeamUser{}).
		Joins("INNER JOIN users ON users.id = team_users.user_id").
		Where("team_users.team_id = ? AND users.email = ?", teamID, email).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// UpdateSettings updates the display name and alerting integrations of a team.
func (s *RepositoryImpl) UpdateSettings(ctx context.Context, team *entities.Team) error {
	result := s.db.WithContext(ctx).
		Model(&entities.Team{}).
		Where("id = ?", team.ID).
		Select(
			"display_name",
			"slack_webhook_url",
			"discord_webhook_url",
			"telegram_chat_id",
			"datadog_webhook_url",
			"new_relic_webhook_url",
			"microsoft_teams_webhook_url",
			"webhook_url",
		).
		Updates(team)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) UpsertInvite(ctx context.Context, invite *entities.TeamInvite) error {
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "team_id"}, {Name: "email"}},
		DoUpdates: clause.AssignmentColumns([]string{"role", "invited_by_user_id", "updated_at"}),
	}).Create(invite).Error
}

func (s *RepositoryImpl) GetInvites(ctx context.Context, teamID uint) (*[]entities.TeamInvite, error) {
	var invites []entities.TeamInvite
	err := s.db.WithContext(ctx).
		Where(entities.TeamInvite{TeamID: teamID}).
		Order("created_at desc").
		Find(&invites).Error

	return &invites, err
}

func (s *RepositoryImpl) GetInvite(ctx context.Context, teamID uint, email string) (*entities.TeamInvite, error) {
	var invite entities.TeamInvite
	if err := s.db.WithContext(ctx).
		Where("team_id = ? AND email = ?", teamID, strings.ToLower(email)).
		First(&invite).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInviteNotFound
		}

		return nil, err
	}

	return &invite, nil
}

func (s *RepositoryImpl) DeleteInvite(ctx context.Context, teamID uint, email string) error {
	result := s.db.WithContext(ctx).
		Where("team_id = ? AND email = ?", teamID, strings.ToLower(email)).
		Delete(&entities.TeamInvite{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInviteNotFound
	}

	return nil
}
