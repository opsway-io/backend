package models

import "github.com/opsway-io/backend/internal/team"

type Team struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Logo      string `json:"logo"`
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
