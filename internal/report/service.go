package report

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/datatypes"
)

type Service interface {
	GetResportsByTeam(ctx context.Context, teamID uint) (*[]entities.Report, error)
	CreateReport(ctx context.Context, teamID uint, reportType string, reportData entities.ReportData) error
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
func (s *ServiceImpl) CreateReport(ctx context.Context, teamID uint, reportType string, reportData entities.ReportData) error {
	reportTypeEnum, err := entities.ReportFrom(reportType)
	if err != nil {
		return err
	}

	return s.repository.Create(ctx, &entities.Report{
		TeamID: teamID,
		Type:   reportTypeEnum,
		Report: datatypes.NewJSONType(reportData),
	})
}
