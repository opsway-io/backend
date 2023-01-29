package team

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrNotFound          = errors.New("team not found")
	ErrUserNotFound      = errors.New("team user not found")
	ErrNameAlreadyExists = errors.New("team name already exists")
)

type Repository interface {
	GetByID(ctx context.Context, teamId uint) (*entities.Team, error)
	GetUsersByID(ctx context.Context, teamId uint, offset *int, limit *int, query *string) (*[]TeamUser, error)
	GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error)
	GetTeamsAndRoleByUserID(ctx context.Context, userID uint) (*[]TeamAndRole, error)
	UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.TeamRole) error
	UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error
	CreateWithOwnerUserID(ctx context.Context, team *entities.Team, ownerUserID uint) error
	Delete(ctx context.Context, id uint) error
	Update(ctx context.Context, team *entities.Team) error
	RemoveUser(ctx context.Context, teamID, userID uint) error
	IsNameAvailable(ctx context.Context, name string) (bool, error)
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
		Joins("INNER JOIN users as u ON u.id = tu.user_id").
		Scopes(
			postgres.Paginated(offset, limit),
			postgres.IncludeTotalCount("total_count"),
			postgres.Search([]string{"u.name", "u.display_name", "u.email"}, query),
		).
		Where("tu.team_id = ?", teamId).
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

func (s *RepositoryImpl) UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.TeamRole) error {
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

func (s *RepositoryImpl) RemoveUser(ctx context.Context, teamID, userID uint) error {
	result := s.db.WithContext(ctx).Delete(&entities.TeamUser{}, entities.TeamUser{
		TeamID: teamID,
		UserID: userID,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
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
