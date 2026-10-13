package statuspage

import (
	"context"
	"time"
)

// noData marks a day without any checks in the daily uptime series.
const noData = -1.0

type checkStats struct {
	hasLatest bool
	latestOK  bool
	// Uptime percentage over the whole period, nil when there are no checks.
	uptime *float64
	// Daily uptime percentages, oldest first, last element is today.
	daily []float64
}

type latestRow struct {
	MonitorID uint64 `gorm:"column:monitor_id"`
	LatestOK  uint8  `gorm:"column:latest_ok"`
}

type dailyRow struct {
	MonitorID uint64    `gorm:"column:monitor_id"`
	Day       time.Time `gorm:"column:day"`
	Total     uint64    `gorm:"column:total"`
	OK        uint64    `gorm:"column:ok"`
}

// A check is considered successful when it got a non-error HTTP status code.
const checkOKExpr = "(status_code >= 200 AND status_code < 400)"

func truncateDay(t time.Time) time.Time {
	t = t.UTC()

	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// monitorCheckStats loads the latest check result and the daily uptime of the
// last UptimeDays days for the given monitors from ClickHouse.
func (s *ServiceImpl) monitorCheckStats(ctx context.Context, teamID uint, monitorIDs []uint, now time.Time) (map[uint]checkStats, error) {
	start := truncateDay(now).AddDate(0, 0, -(UptimeDays - 1))

	if len(monitorIDs) == 0 || s.checks == nil {
		return buildCheckStats(monitorIDs, nil, nil, start), nil
	}

	ids := make([]uint64, len(monitorIDs))
	for i, id := range monitorIDs {
		ids[i] = uint64(id)
	}

	var latest []latestRow
	if err := s.checks.WithContext(ctx).
		Table("checks").
		Select("monitor_id, toUInt8(argMax("+checkOKExpr+", created_at)) AS latest_ok").
		Where("team_id = ? AND monitor_id IN ? AND created_at >= ?", uint64(teamID), ids, start).
		Group("monitor_id").
		Scan(&latest).Error; err != nil {
		return nil, err
	}

	var daily []dailyRow
	if err := s.checks.WithContext(ctx).
		Table("checks").
		Select("monitor_id, toStartOfDay(created_at, 'UTC') AS day, count() AS total, countIf("+checkOKExpr+") AS ok").
		Where("team_id = ? AND monitor_id IN ? AND created_at >= ?", uint64(teamID), ids, start).
		Group("monitor_id, day").
		Scan(&daily).Error; err != nil {
		return nil, err
	}

	return buildCheckStats(monitorIDs, latest, daily, start), nil
}

// buildCheckStats folds the raw query rows into per-monitor stats covering
// UptimeDays days starting at start. Rows outside the window are ignored.
func buildCheckStats(monitorIDs []uint, latest []latestRow, daily []dailyRow, start time.Time) map[uint]checkStats {
	start = truncateDay(start)

	type counts struct{ total, ok uint64 }
	perDay := map[uint]map[int]counts{}

	for _, r := range daily {
		idx := int(truncateDay(r.Day).Sub(start).Hours() / 24)
		if idx < 0 || idx >= UptimeDays {
			continue
		}

		id := uint(r.MonitorID)
		if perDay[id] == nil {
			perDay[id] = map[int]counts{}
		}

		c := perDay[id][idx]
		perDay[id][idx] = counts{total: c.total + r.Total, ok: c.ok + r.OK}
	}

	latestByID := map[uint]bool{}
	for _, r := range latest {
		latestByID[uint(r.MonitorID)] = r.LatestOK != 0
	}

	res := make(map[uint]checkStats, len(monitorIDs))
	for _, id := range monitorIDs {
		st := checkStats{daily: make([]float64, UptimeDays)}
		st.latestOK, st.hasLatest = latestByID[id]

		var total, ok uint64
		for i := 0; i < UptimeDays; i++ {
			c, found := perDay[id][i]
			if !found || c.total == 0 {
				st.daily[i] = noData

				continue
			}

			st.daily[i] = float64(c.ok) * 100 / float64(c.total)
			total += c.total
			ok += c.ok
		}

		if total > 0 {
			u := float64(ok) * 100 / float64(total)
			st.uptime = &u
		}

		res[id] = st
	}

	return res
}
