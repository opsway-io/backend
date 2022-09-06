package models

import "github.com/opsway-io/backend/internal/team"

type Team struct {
	ID        int    `json:"id" validate:"required,numeric,gte=0"`
	Name      string `json:"name" validate:"required,min=1,max=255"`
	Logo      string `json:"logo" validate:"required,url"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

func TeamToResponse(team team.Team) Team {
	return Team{
		ID:        team.ID,
		Name:      team.Name,
		Logo:      team.Logo,
		CreatedAt: team.CreatedAt.Unix(),
		UpdatedAt: team.UpdatedAt.Unix(),
	}
}

func RequestToTeam(req Team) team.Team {
	return team.Team{
		Name: req.Name,
		Logo: req.Logo,
	}
}
