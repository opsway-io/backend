package monitor

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// MigrateToSteps converts monitors created with the legacy single-request model
// into 1-step monitors: the request from the monitor settings becomes step 1 and
// all assertions without a step are attached to it (keeping their IDs, which
// incidents reference). It is idempotent and safe to run on every start-up.
func MigrateToSteps(ctx context.Context, db *gorm.DB) (int, error) {
	var monitors []entities.Monitor
	if err := db.WithContext(ctx).Preload("Settings").Where(
		"NOT EXISTS (SELECT 1 FROM monitor_steps s WHERE s.monitor_id = monitors.id)",
	).Find(&monitors).Error; err != nil {
		return 0, errors.Wrap(err, "failed to find monitors without steps")
	}

	migrated := 0

	for _, m := range monitors {
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Re-check inside the transaction to stay idempotent with concurrent starts
			var count int64
			if err := tx.Model(&entities.MonitorStep{}).Where("monitor_id = ?", m.ID).Count(&count).Error; err != nil {
				return err
			}

			if count > 0 {
				return nil
			}

			method := m.Settings.Method
			if method == "" {
				method = "GET"
			}

			step := entities.MonitorStep{
				MonitorID: m.ID,
				Position:  0,
				Name:      "Step 1",
				Method:    method,
				URL:       m.Settings.URL,
				Headers:   m.Settings.Headers,
				Body:      m.Settings.Body,
			}

			if step.Body.Type == "" {
				step.Body.Type = "NONE"
			}

			if err := tx.Omit("Assertions").Create(&step).Error; err != nil {
				return err
			}

			if err := tx.Model(&entities.MonitorAssertion{}).Where(
				"monitor_id = ? AND step_id IS NULL", m.ID,
			).Update("step_id", step.ID).Error; err != nil {
				return err
			}

			if m.Settings.Auth.Method == "" {
				if err := tx.Model(&entities.MonitorSettings{}).Where(
					"monitor_id = ?", m.ID,
				).Update("auth_method", entities.MonitorAuthMethodNone).Error; err != nil {
					return err
				}
			}

			migrated++

			return nil
		})
		if err != nil {
			return migrated, errors.Wrapf(err, "failed to migrate monitor %d", m.ID)
		}
	}

	return migrated, nil
}
