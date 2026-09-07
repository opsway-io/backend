package user

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
	ErrNotFound           = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("user with same email already exists")
)

type Repository interface {
	GetUserByID(ctx context.Context, userID uint) (*entities.User, error)
	GetUserAndTeamsByUserID(ctx context.Context, userID uint) (*entities.User, error)
	GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error)
	Create(ctx context.Context, user *entities.User) error
	Update(ctx context.Context, user *entities.User) error
	Delete(ctx context.Context, id uint) error

	GetNotificationRules(ctx context.Context, userID uint) ([]entities.UserNotificationRule, error)
	SetNotificationRules(ctx context.Context, userID uint, rules []entities.UserNotificationRule) error
}

type RepositoryImpl struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &RepositoryImpl{db: db}
}

func (r *RepositoryImpl) GetUserByID(ctx context.Context, userID uint) (*entities.User, error) {
	var user entities.User
	err := r.db.WithContext(ctx).First(&user, userID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) GetUserAndTeamsByUserID(ctx context.Context, userID uint) (*entities.User, error) {
	var user entities.User
	if err := s.db.WithContext(ctx).Preload("Teams").Where(entities.User{ID: userID}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) GetUserAndTeamsByEmailAddress(ctx context.Context, email string) (*entities.User, error) {
	var user entities.User
	if err := s.db.WithContext(ctx).Preload("Teams").Where(entities.User{Email: strings.ToLower(email)}).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &user, nil
}

func (s *RepositoryImpl) Create(ctx context.Context, user *entities.User) error {
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		if postgres.IsDuplicateEntryError(err) {
			return ErrEmailAlreadyExists
		}

		return err
	}

	return nil
}

func (s *RepositoryImpl) Update(ctx context.Context, user *entities.User) error {
	result := s.db.WithContext(ctx).Model(user).Updates(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *RepositoryImpl) Delete(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Select(clause.Associations).Delete(&entities.User{
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

func (s *RepositoryImpl) GetNotificationRules(ctx context.Context, userID uint) ([]entities.UserNotificationRule, error) {
	var rules []entities.UserNotificationRule
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("delay asc").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (s *RepositoryImpl) SetNotificationRules(ctx context.Context, userID uint, rules []entities.UserNotificationRule) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&entities.UserNotificationRule{}).Error; err != nil {
			return err
		}
		if len(rules) > 0 {
			if err := tx.Create(&rules).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

