package team

import (
	"context"
	"errors"

	"github.com/opsway-io/backend/internal/connectors/postgres"
	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/gorm"
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
	RemoveUser(ctx context.Context, teamID, userID uint) error
	UpdateDisplayName(ctx context.Context, teamID uint, displayName string) error
	UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.Role) error
	Create(ctx context.Context, team *entities.Team) error
	Delete(ctx context.Context, id uint) error
	Update(ctx context.Context, team *entities.Team) error
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
	Role       entities.Role
	TotalCount int
}

func (s *RepositoryImpl) GetUsersByID(ctx context.Context, teamId uint, offset *int, limit *int, query *string) (*[]TeamUser, error) {
	var users []TeamUser

	s.db.WithContext(ctx).
		Select("u.*, tr.role").
		Table("team_users as tu").
		Joins("INNER JOIN team_roles AS tr ON tr.user_id = tu.user_id").
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

func (s *RepositoryImpl) Create(ctx context.Context, team *entities.Team) error {
	if err := s.db.WithContext(ctx).Create(team).Error; err != nil {
		if errors.As(err, &postgres.ErrDuplicateEntry) {
			return ErrNameAlreadyExists
		}

		return err
	}

	return nil
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

func (s *RepositoryImpl) UpdateUserRole(ctx context.Context, teamID, userID uint, role entities.Role) error {
	result := s.db.WithContext(ctx).Model(&entities.TeamRole{}).Where(entities.TeamRole{
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
	result := s.db.WithContext(ctx).Delete(&entities.Team{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) GetUserRole(ctx context.Context, teamID, userID uint) (*entities.TeamRole, error) {
	var userRole entities.TeamRole
	if err := s.db.WithContext(ctx).Where(
		entities.TeamRole{
			TeamID: teamID,
			UserID: userID,
		},
	).First(&userRole).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &userRole, nil
}

func (s *RepositoryImpl) RemoveUser(ctx context.Context, teamID, userID uint) error {
	result := s.db.WithContext(ctx).Delete(&entities.TeamRole{}, entities.TeamRole{
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
