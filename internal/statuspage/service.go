package statuspage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/opsway-io/backend/internal/notification/email"
	"gorm.io/gorm"
)

var (
	ErrNotFound         = errors.New("status page not found")
	ErrDomainTaken      = errors.New("domain already in use")
	ErrInvalidMonitor   = errors.New("monitor does not belong to team")
	ErrSubscriberExists = errors.New("subscriber already exists")
	ErrInvalidToken     = errors.New("invalid verification token")
	ErrPasswordRequired = errors.New("password required for private status page")
)

type Service interface {
	GetByTeamID(ctx context.Context, teamID uint) (*[]entities.StatusPage, error)
	GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.StatusPage, error)
	GetByDomain(ctx context.Context, domain string) (*entities.StatusPage, error)
	Create(ctx context.Context, page *entities.StatusPage) error
	// Update replaces all settings, groups and monitors of the status page.
	Update(ctx context.Context, page *entities.StatusPage) error
	Delete(ctx context.Context, teamID, id uint) error

	// GetPublicView assembles the public, read-only representation of a status page.
	GetPublicView(ctx context.Context, page *entities.StatusPage) (*PublicView, error)
	// GetOverallStatus is a cheap summary used by the badge / widget.
	GetOverallStatus(ctx context.Context, page *entities.StatusPage) (OverallStatus, error)

	Subscribe(ctx context.Context, page *entities.StatusPage, channel entities.StatusPageSubscriberChannel, emailAddr, destination string) error
	VerifySubscriber(ctx context.Context, page *entities.StatusPage, token string) error
}

type ServiceImpl struct {
	db          *gorm.DB
	checks      *gorm.DB
	emailSender email.Sender
	now         func() time.Time
}

// NewService creates the status page service. checksDB is the ClickHouse
// connection holding monitor check results (used for status and uptime bars).
func NewService(db *gorm.DB, checksDB *gorm.DB, emailSender email.Sender) Service {
	return &ServiceImpl{
		db:          db,
		checks:      checksDB,
		emailSender: emailSender,
		now:         time.Now,
	}
}

func NormalizeDomain(domain string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
}

func (s *ServiceImpl) withGroups(ctx context.Context) *gorm.DB {
	return s.db.WithContext(ctx).Preload("Groups", func(db *gorm.DB) *gorm.DB {
		return db.Order(`"order" ASC, id ASC`)
	})
}

func (s *ServiceImpl) GetByTeamID(ctx context.Context, teamID uint) (*[]entities.StatusPage, error) {
	var pages []entities.StatusPage
	if err := s.withGroups(ctx).
		Where("team_id = ?", teamID).
		Order("name ASC, id ASC").
		Find(&pages).Error; err != nil {
		return nil, err
	}

	return &pages, nil
}

func (s *ServiceImpl) GetByTeamIDAndID(ctx context.Context, teamID, id uint) (*entities.StatusPage, error) {
	var page entities.StatusPage
	if err := s.withGroups(ctx).
		Where("team_id = ? AND id = ?", teamID, id).
		First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &page, nil
}

func (s *ServiceImpl) GetByDomain(ctx context.Context, domain string) (*entities.StatusPage, error) {
	var page entities.StatusPage
	if err := s.withGroups(ctx).
		Where("domain = ?", NormalizeDomain(domain)).
		First(&page).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}

		return nil, err
	}

	return &page, nil
}

func (s *ServiceImpl) domainTaken(tx *gorm.DB, domain string, exceptID uint) (bool, error) {
	var count int64
	if err := tx.Model(&entities.StatusPage{}).
		Where("domain = ? AND id <> ?", domain, exceptID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func validateMonitors(tx *gorm.DB, page *entities.StatusPage) error {
	ids := page.AllMonitorIDs()
	if len(ids) == 0 {
		return nil
	}

	var count int64
	if err := tx.Model(&entities.Monitor{}).
		Where("team_id = ? AND id IN ?", page.TeamID, ids).
		Count(&count).Error; err != nil {
		return err
	}

	if int(count) != len(ids) {
		return ErrInvalidMonitor
	}

	return nil
}

func (s *ServiceImpl) Create(ctx context.Context, page *entities.StatusPage) error {
	page.Domain = NormalizeDomain(page.Domain)
	if page.Layout == "" {
		page.Layout = string(entities.StatusPageLayoutStats)
	}
	if page.MonitorIDs == nil {
		page.MonitorIDs = pq.Int64Array{}
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		taken, err := s.domainTaken(tx, page.Domain, 0)
		if err != nil {
			return err
		}
		if taken {
			return ErrDomainTaken
		}

		if err := validateMonitors(tx, page); err != nil {
			return err
		}

		return tx.Create(page).Error
	})
}

func (s *ServiceImpl) Update(ctx context.Context, page *entities.StatusPage) error {
	page.Domain = NormalizeDomain(page.Domain)
	if page.MonitorIDs == nil {
		page.MonitorIDs = pq.Int64Array{}
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing entities.StatusPage
		if err := tx.Where("team_id = ? AND id = ?", page.TeamID, page.ID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}

			return err
		}

		taken, err := s.domainTaken(tx, page.Domain, page.ID)
		if err != nil {
			return err
		}
		if taken {
			return ErrDomainTaken
		}

		if err := validateMonitors(tx, page); err != nil {
			return err
		}

		if !page.IsPrivate {
			page.PasswordHash = nil
		} else if page.PasswordHash == nil {
			// Keep existing password when none is supplied.
			page.PasswordHash = existing.PasswordHash
		}

		if page.IsPrivate && page.PasswordHash == nil {
			return ErrPasswordRequired
		}

		if err := tx.Model(&entities.StatusPage{}).Where("id = ?", page.ID).Updates(map[string]any{
			"name":                   page.Name,
			"domain":                 page.Domain,
			"logo_url":               page.LogoURL,
			"logo_link":              page.LogoLink,
			"favicon_url":            page.FaviconURL,
			"layout":                 page.Layout,
			"custom_css":             page.CustomCSS,
			"header_html":            page.HeaderHTML,
			"footer_html":            page.FooterHTML,
			"custom_components_html": page.CustomComponentsHTML,
			"show_branding":          page.ShowBranding,
			"is_private":             page.IsPrivate,
			"password_hash":          page.PasswordHash,
			"support_url":            page.SupportURL,
			"monitor_ids":            page.MonitorIDs,
			"updated_at":             s.now(),
		}).Error; err != nil {
			return err
		}

		if err := tx.Where("status_page_id = ?", page.ID).Delete(&entities.StatusPageGroup{}).Error; err != nil {
			return err
		}

		for i := range page.Groups {
			page.Groups[i].ID = 0
			page.Groups[i].StatusPageID = page.ID
			if page.Groups[i].MonitorIDs == nil {
				page.Groups[i].MonitorIDs = pq.Int64Array{}
			}
		}

		if len(page.Groups) > 0 {
			if err := tx.Create(&page.Groups).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *ServiceImpl) Delete(ctx context.Context, teamID, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var page entities.StatusPage
		if err := tx.Where("team_id = ? AND id = ?", teamID, id).First(&page).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}

			return err
		}

		return tx.Select("Groups", "Subscribers").Delete(&page).Error
	})
}

func (s *ServiceImpl) Subscribe(
	ctx context.Context,
	page *entities.StatusPage,
	channel entities.StatusPageSubscriberChannel,
	emailAddr, destination string,
) error {
	token, err := randomToken()
	if err != nil {
		return err
	}

	sub := &entities.StatusPageSubscriber{
		StatusPageID:      page.ID,
		Channel:           channel,
		Email:             strings.ToLower(strings.TrimSpace(emailAddr)),
		Destination:       strings.TrimSpace(destination),
		VerificationToken: token,
	}

	// Webhook and Slack subscriptions are active immediately (the frontend
	// tells the user so); email subscriptions require double opt-in.
	if channel != entities.StatusPageSubscriberChannelEmail {
		now := s.now()
		sub.VerifiedAt = &now
	}

	var existing entities.StatusPageSubscriber
	err = s.db.WithContext(ctx).Where(
		"status_page_id = ? AND channel = ? AND email = ? AND destination = ?",
		sub.StatusPageID, sub.Channel, sub.Email, sub.Destination,
	).First(&existing).Error

	switch {
	case err == nil:
		if existing.VerifiedAt != nil {
			// Do not leak whether an address is subscribed; treat as success.
			return nil
		}
		// Re-send verification with the existing token.
		sub = &existing
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := s.db.WithContext(ctx).Create(sub).Error; err != nil {
			return err
		}
	default:
		return err
	}

	if sub.Channel == entities.StatusPageSubscriberChannelEmail && sub.VerifiedAt == nil {
		link := fmt.Sprintf("https://%s/subscribe/verify/%s", page.Domain, sub.VerificationToken)

		return s.emailSender.Send(ctx, sub.Email, sub.Email, &verifySubscriptionTemplate{
			PageName: page.Name,
			Link:     link,
		})
	}

	return nil
}

func (s *ServiceImpl) VerifySubscriber(ctx context.Context, page *entities.StatusPage, token string) error {
	now := s.now()

	result := s.db.WithContext(ctx).
		Model(&entities.StatusPageSubscriber{}).
		Where("status_page_id = ? AND verification_token = ?", page.ID, token).
		Update("verified_at", gorm.Expr("COALESCE(verified_at, ?)", now))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInvalidToken
	}

	return nil
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

type verifySubscriptionTemplate struct {
	PageName string
	Link     string
}

func (t *verifySubscriptionTemplate) Subject() string {
	return fmt.Sprintf("Confirm your subscription to %s", t.PageName)
}

func (t *verifySubscriptionTemplate) HTML() string {
	return fmt.Sprintf(
		`<p>Please confirm your subscription to status updates for <b>%s</b>.</p><p><a href="%s">Confirm subscription</a></p>`,
		htmlEscape(t.PageName), htmlEscape(t.Link),
	)
}

func (t *verifySubscriptionTemplate) PlainText() string {
	return fmt.Sprintf("Please confirm your subscription to status updates for %s: %s", t.PageName, t.Link)
}
