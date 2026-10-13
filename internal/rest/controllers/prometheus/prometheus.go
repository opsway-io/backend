package prometheus

import (
	"context"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type OpswayCollector struct {
	db                  *gorm.DB
	ch                  *gorm.DB
	monitorsDesc        *prometheus.Desc
	activeIncidentsDesc *prometheus.Desc
	checksDesc          *prometheus.Desc
}

func NewOpswayCollector(db *gorm.DB, ch *gorm.DB) *OpswayCollector {
	return &OpswayCollector{
		db: db,
		ch: ch,
		monitorsDesc: prometheus.NewDesc(
			"opsway_monitors_total",
			"Total number of configured monitors in the system.",
			nil, nil,
		),
		activeIncidentsDesc: prometheus.NewDesc(
			"opsway_active_incidents",
			"Current number of active, unresolved incidents.",
			nil, nil,
		),
		checksDesc: prometheus.NewDesc(
			"opsway_checks_processed_total",
			"Total number of processed HTTP checks.",
			nil, nil,
		),
	}
}

func (c *OpswayCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.monitorsDesc
	ch <- c.activeIncidentsDesc
	ch <- c.checksDesc
}

func (c *OpswayCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Fetch monitors count
	var monitorsCount int64
	if err := c.db.WithContext(ctx).Table("monitors").Count(&monitorsCount).Error; err == nil {
		ch <- prometheus.MustNewConstMetric(c.monitorsDesc, prometheus.GaugeValue, float64(monitorsCount))
	}

	// Fetch active incidents
	var activeIncidents int64
	if err := c.db.WithContext(ctx).Table("incidents").Where("resolved = ?", false).Count(&activeIncidents).Error; err == nil {
		ch <- prometheus.MustNewConstMetric(c.activeIncidentsDesc, prometheus.GaugeValue, float64(activeIncidents))
	}

	// Fetch total checks processed
	var checksCount int64
	if err := c.ch.WithContext(ctx).Table("checks").Count(&checksCount).Error; err == nil {
		ch <- prometheus.MustNewConstMetric(c.checksDesc, prometheus.CounterValue, float64(checksCount))
	}
}

func Register(e *echo.Echo, logger *logrus.Entry, db *gorm.DB, ch *gorm.DB) {
	if db == nil || ch == nil {
		logger.Warn("Prometheus metrics collector registration skipped (database connection is nil)")
		return
	}
	collector := NewOpswayCollector(db, ch)
	prometheus.MustRegister(collector)

	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))
	logger.Info("Registered Prometheus /metrics endpoint")
}
