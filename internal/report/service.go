package report

import (
	"context"

	"github.com/opsway-io/backend/internal/entities"
	"gorm.io/datatypes"
)

type Service interface {
	GetResportsByTeam(ctx context.Context, teamID uint) (*[]entities.Report, error)
	GetByID(ctx context.Context, id uint) (*entities.Report, error)
	CreateReport(ctx context.Context, teamID uint, reportType string, reportData entities.ReportData) (*entities.Report, error)
	Update(ctx context.Context, rep *entities.Report) error
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
func (s *ServiceImpl) GetByID(ctx context.Context, id uint) (*entities.Report, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, rep *entities.Report) error {
	return s.repository.Update(ctx, rep)
}

func (s *ServiceImpl) CreateReport(ctx context.Context, teamID uint, reportType string, reportData entities.ReportData) (*entities.Report, error) {
	reportTypeEnum, err := entities.ReportFrom(reportType)
	if err != nil {
		return nil, err
	}

	rep := &entities.Report{
		TeamID: teamID,
		Type:   reportTypeEnum,
		Status: entities.ReportStatusPending,
		Report: datatypes.NewJSONType(reportData),
	}

	err = s.repository.Create(ctx, rep)
	if err != nil {
		return nil, err
	}
	return rep, nil
}
