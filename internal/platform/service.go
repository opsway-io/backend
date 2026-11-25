package platform

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrInvalidProduct    = errors.New("invalid product slug")
	ErrInvalidVisibility = errors.New("invalid visibility mode")
	ErrUserNotFound      = errors.New("user not found")
	ErrLastPlatformOwner = errors.New("cannot remove the last platform owner")
)

type AllowedUserSummary struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type ProductFlagDTO struct {
	Product        entities.ProductSlug       `json:"product"`
	Visibility     entities.ProductVisibility `json:"visibility"`
	AllowedUserIDs []uint                     `json:"allowedUserIds"`
	AllowedUsers   []AllowedUserSummary       `json:"allowedUsers"`
	AllowedTeamIDs []uint                     `json:"allowedTeamIds"`
	Description    string                     `json:"description"`
	UpdatedAt      time.Time                  `json:"updatedAt"`
}

type UpdateFlagInput struct {
	Visibility     entities.ProductVisibility
	AllowedUserIDs []uint
	AllowedEmails  []string
	AllowedTeamIDs []uint
	Description    *string
}

type OwnerDTO struct {
	ID          uint    `json:"id"`
	Name        string  `json:"name"`
	DisplayName *string `json:"displayName,omitempty"`
	Email       string  `json:"email"`
}

type Service interface {
	EnsureDefaults(ctx context.Context) error
	BootstrapOwners(ctx context.Context, emails []string) error
	IsPlatformOwner(ctx context.Context, userID uint) (bool, error)
	IsProductEnabledForUser(ctx context.Context, product entities.ProductSlug, userID uint, teamID *uint) (bool, error)
	GetUserProductVisibility(ctx context.Context, userID uint, teamID *uint) (bool, map[entities.ProductSlug]bool, error)
	ListFlags(ctx context.Context) ([]ProductFlagDTO, error)
	UpdateFlag(ctx context.Context, product entities.ProductSlug, input UpdateFlagInput, updatedBy uint) (*ProductFlagDTO, error)
	ListOwners(ctx context.Context) ([]OwnerDTO, error)
	AddOwnerByEmail(ctx context.Context, email string) (*OwnerDTO, error)
	RemoveOwner(ctx context.Context, targetUserID uint, actorUserID uint) error
}

type service struct {
	db     *gorm.DB
	logger *logrus.Entry
}

func NewService(db *gorm.DB, logger *logrus.Entry) Service {
	return &service{
		db:     db,
		logger: logger,
	}
}

func defaultDescription(p entities.ProductSlug) string {
	switch p {
	case entities.ProductMonitoring:
		return "Uptime monitors, alerting, incidents, heartbeats, maintenance, status pages, and reports"
	case entities.ProductLogSearch:
		return "Semantic and SQL log explorer"
	case entities.ProductManagedAgent:
		return "Autonomous AI SRE agents, investigations, and MCP gateway"
	case entities.ProductPantomath:
		return "Pantomath service intelligence and deployment tracking"
	default:
		return ""
	}
}

func (s *service) EnsureDefaults(ctx context.Context) error {
	for _, p := range entities.AllProducts() {
		flag := entities.ProductFeatureFlag{
			Product:        p,
			Visibility:     entities.ProductVisibilityEnabled,
			AllowedUserIDs: []uint{},
			AllowedTeamIDs: []uint{},
			Description:    defaultDescription(p),
		}
		if err := s.db.WithContext(ctx).
			Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "product"}}, DoNothing: true}).
			Create(&flag).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *service) BootstrapOwners(ctx context.Context, emails []string) error {
	for _, raw := range emails {
		email := strings.ToLower(strings.TrimSpace(raw))
		if email == "" {
			continue
		}
		if err := s.db.WithContext(ctx).
			Model(&entities.User{}).
			Where("LOWER(email) = ?", email).
			Update("is_platform_owner", true).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *service) IsPlatformOwner(ctx context.Context, userID uint) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	var u entities.User
	if err := s.db.WithContext(ctx).Select("id", "is_platform_owner").First(&u, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return u.IsPlatformOwner, nil
}

func evaluateFlag(flag *entities.ProductFeatureFlag, isOwner bool, userID uint, teamID *uint) bool {
	if flag == nil {
		return true
	}
	switch flag.Visibility {
	case entities.ProductVisibilityDisabled:
		return false
	case entities.ProductVisibilityOwnerOnly:
		return isOwner
	case entities.ProductVisibilityTargeted:
		if isOwner {
			return true
		}
		if userID != 0 && slices.Contains(flag.AllowedUserIDs, userID) {
			return true
		}
		if teamID != nil && *teamID != 0 && slices.Contains(flag.AllowedTeamIDs, *teamID) {
			return true
		}
		return false
	case entities.ProductVisibilityEnabled:
		return true
	default:
		return true
	}
}

func (s *service) IsProductEnabledForUser(ctx context.Context, product entities.ProductSlug, userID uint, teamID *uint) (bool, error) {
	if !entities.IsValidProduct(product) {
		return false, ErrInvalidProduct
	}
	isOwner, err := s.IsPlatformOwner(ctx, userID)
	if err != nil {
		return false, err
	}
	var flag entities.ProductFeatureFlag
	if err := s.db.WithContext(ctx).Where("product = ?", product).First(&flag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}
	return evaluateFlag(&flag, isOwner, userID, teamID), nil
}

func (s *service) GetUserProductVisibility(ctx context.Context, userID uint, teamID *uint) (bool, map[entities.ProductSlug]bool, error) {
	isOwner, err := s.IsPlatformOwner(ctx, userID)
	if err != nil {
		return false, nil, err
	}

	var flags []entities.ProductFeatureFlag
	if err := s.db.WithContext(ctx).Find(&flags).Error; err != nil {
		return false, nil, err
	}

	byProduct := make(map[entities.ProductSlug]*entities.ProductFeatureFlag, len(flags))
	for i := range flags {
		byProduct[flags[i].Product] = &flags[i]
	}

	result := make(map[entities.ProductSlug]bool, len(entities.AllProducts()))
	for _, p := range entities.AllProducts() {
		result[p] = evaluateFlag(byProduct[p], isOwner, userID, teamID)
	}
	return isOwner, result, nil
}

func (s *service) toDTO(ctx context.Context, flag *entities.ProductFeatureFlag) (*ProductFlagDTO, error) {
	allowedUsers := make([]AllowedUserSummary, 0, len(flag.AllowedUserIDs))
	if len(flag.AllowedUserIDs) > 0 {
		var users []entities.User
		if err := s.db.WithContext(ctx).
			Select("id", "name", "email").
			Where("id IN ?", flag.AllowedUserIDs).
			Find(&users).Error; err != nil {
			return nil, err
		}
		for _, u := range users {
			allowedUsers = append(allowedUsers, AllowedUserSummary{
				ID:    u.ID,
				Name:  u.Name,
				Email: u.Email,
			})
		}
	}

	userIDs := flag.AllowedUserIDs
	if userIDs == nil {
		userIDs = []uint{}
	}
	teamIDs := flag.AllowedTeamIDs
	if teamIDs == nil {
		teamIDs = []uint{}
	}

	return &ProductFlagDTO{
		Product:        flag.Product,
		Visibility:     flag.Visibility,
		AllowedUserIDs: userIDs,
		AllowedUsers:   allowedUsers,
		AllowedTeamIDs: teamIDs,
		Description:    flag.Description,
		UpdatedAt:      flag.UpdatedAt,
	}, nil
}

func (s *service) ListFlags(ctx context.Context) ([]ProductFlagDTO, error) {
	if err := s.EnsureDefaults(ctx); err != nil {
		return nil, err
	}
	var flags []entities.ProductFeatureFlag
	if err := s.db.WithContext(ctx).Order("id ASC").Find(&flags).Error; err != nil {
		return nil, err
	}
	out := make([]ProductFlagDTO, 0, len(flags))
	for i := range flags {
		dto, err := s.toDTO(ctx, &flags[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *dto)
	}
	return out, nil
}

func (s *service) UpdateFlag(ctx context.Context, product entities.ProductSlug, input UpdateFlagInput, updatedBy uint) (*ProductFlagDTO, error) {
	if !entities.IsValidProduct(product) {
		return nil, ErrInvalidProduct
	}
	if !entities.IsValidVisibility(input.Visibility) {
		return nil, ErrInvalidVisibility
	}

	// Merge explicit user IDs with any emails resolved to user IDs.
	userIDSet := make(map[uint]struct{}, len(input.AllowedUserIDs)+len(input.AllowedEmails))
	for _, id := range input.AllowedUserIDs {
		if id > 0 {
			userIDSet[id] = struct{}{}
		}
	}
	if len(input.AllowedEmails) > 0 {
		normalized := make([]string, 0, len(input.AllowedEmails))
		for _, e := range input.AllowedEmails {
			trimmed := strings.ToLower(strings.TrimSpace(e))
			if trimmed != "" {
				normalized = append(normalized, trimmed)
			}
		}
		if len(normalized) > 0 {
			var matched []entities.User
			if err := s.db.WithContext(ctx).
				Select("id", "email").
				Where("LOWER(email) IN ?", normalized).
				Find(&matched).Error; err != nil {
				return nil, err
			}
			if len(matched) == 0 && len(input.AllowedUserIDs) == 0 && len(normalized) > 0 {
				return nil, ErrUserNotFound
			}
			for _, u := range matched {
				userIDSet[u.ID] = struct{}{}
			}
		}
	}

	mergedUserIDs := make([]uint, 0, len(userIDSet))
	for id := range userIDSet {
		mergedUserIDs = append(mergedUserIDs, id)
	}
	slices.Sort(mergedUserIDs)

	teamIDs := input.AllowedTeamIDs
	if teamIDs == nil {
		teamIDs = []uint{}
	}

	var flag entities.ProductFeatureFlag
	err := s.db.WithContext(ctx).Where("product = ?", product).First(&flag).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		flag = entities.ProductFeatureFlag{
			Product:     product,
			Description: defaultDescription(product),
		}
	} else if err != nil {
		return nil, err
	}

	flag.Visibility = input.Visibility
	flag.AllowedUserIDs = mergedUserIDs
	flag.AllowedTeamIDs = teamIDs
	if input.Description != nil {
		flag.Description = *input.Description
	}
	if updatedBy > 0 {
		flag.UpdatedBy = &updatedBy
	}

	if err := s.db.WithContext(ctx).Save(&flag).Error; err != nil {
		return nil, err
	}

	return s.toDTO(ctx, &flag)
}

func (s *service) ListOwners(ctx context.Context) ([]OwnerDTO, error) {
	var users []entities.User
	if err := s.db.WithContext(ctx).
		Select("id", "name", "display_name", "email").
		Where("is_platform_owner = ?", true).
		Order("id ASC").
		Find(&users).Error; err != nil {
		return nil, err
	}
	out := make([]OwnerDTO, len(users))
	for i, u := range users {
		out[i] = OwnerDTO{
			ID:          u.ID,
			Name:        u.Name,
			DisplayName: u.DisplayName,
			Email:       u.Email,
		}
	}
	return out, nil
}

func (s *service) AddOwnerByEmail(ctx context.Context, email string) (*OwnerDTO, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return nil, ErrUserNotFound
	}
	var u entities.User
	if err := s.db.WithContext(ctx).Where("LOWER(email) = ?", normalized).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if !u.IsPlatformOwner {
		if err := s.db.WithContext(ctx).Model(&u).Update("is_platform_owner", true).Error; err != nil {
			return nil, err
		}
	}
	return &OwnerDTO{
		ID:          u.ID,
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Email:       u.Email,
	}, nil
}

func (s *service) RemoveOwner(ctx context.Context, targetUserID uint, actorUserID uint) error {
	var count int64
	if err := s.db.WithContext(ctx).Model(&entities.User{}).Where("is_platform_owner = ?", true).Count(&count).Error; err != nil {
		return err
	}
	if count <= 1 {
		return ErrLastPlatformOwner
	}
	res := s.db.WithContext(ctx).Model(&entities.User{}).Where("id = ?", targetUserID).Update("is_platform_owner", false)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}
