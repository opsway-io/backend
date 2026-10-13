package report

import (
	"context"
	"time"

	"github.com/opsway-io/backend/internal/entities"
	"github.com/pkg/errors"
	"gorm.io/datatypes"
)

type Service interface {
	GetResportsByTeam(ctx context.Context, teamID uint) (*[]entities.Report, error)
	GetByTeamIDAndID(ctx context.Context, teamID, reportID uint) (*entities.Report, error)
	// Generate builds the report data for the given period and stores the report.
	// If gathering data fails the report is stored with status FAILED and the error is returned.
	Generate(ctx context.Context, teamID uint, reportType entities.ReportType, start, end time.Time) (*entities.Report, error)
	Delete(ctx context.Context, teamID, reportID uint) error
}

type ServiceImpl struct {
	repository Repository
}

func NewService(repository Repository) Service {
	return &ServiceImpl{
		repository: repository,
	}
}

func (s *ServiceImpl) GetResportsByTeam(ctx context.Context, teamID uint) (*[]entities.Report, error) {
	return s.repository.GetReportsByTeamID(ctx, teamID)
}

func (s *ServiceImpl) GetByTeamIDAndID(ctx context.Context, teamID, reportID uint) (*entities.Report, error) {
	return s.repository.GetByTeamIDAndID(ctx, teamID, reportID)
}

func (s *ServiceImpl) Delete(ctx context.Context, teamID, reportID uint) error {
	return s.repository.Delete(ctx, teamID, reportID)
}

func (s *ServiceImpl) Generate(ctx context.Context, teamID uint, reportType entities.ReportType, start, end time.Time) (*entities.Report, error) {
	data, genErr := s.collect(ctx, teamID, reportType, start, end)

	rep := &entities.Report{
		TeamID: teamID,
		Type:   reportType,
		Status: entities.ReportStatusCompleted,
		Start:  &start,
		End:    &end,
		Report: datatypes.NewJSONType(data),
	}
	if genErr != nil {
		rep.Status = entities.ReportStatusFailed
		rep.Report = datatypes.NewJSONType(entities.ReportData{})
	}

	if err := s.repository.Create(ctx, rep); err != nil {
		return nil, errors.Wrap(err, "failed to store report")
	}

	return rep, genErr
}

func (s *ServiceImpl) collect(ctx context.Context, teamID uint, reportType entities.ReportType, start, end time.Time) (entities.ReportData, error) {
	var data entities.ReportData

	all := reportType == entities.ReportTypeAll || reportType == entities.ReportTypeCustom

	if all || reportType == entities.ReportTypeUptime {
		uptime, err := s.repository.GetUptime(ctx, teamID, start, end)
		if err != nil {
			return data, errors.Wrap(err, "failed to get uptime")
		}
		data.Uptime = uptime
	}

	if all || reportType == entities.ReportTypePerformance {
		perf, err := s.repository.GetPerformance(ctx, teamID, start, end)
		if err != nil {
			return data, errors.Wrap(err, "failed to get performance")
		}
		data.Performance = perf
	}

	if all || reportType == entities.ReportTypeIncident {
		incidents, err := s.repository.GetIncidentCounts(ctx, teamID, start, end)
		if err != nil {
			return data, errors.Wrap(err, "failed to get incident counts")
		}
		data.Incident = incidents
	}

	return data, nil
}
