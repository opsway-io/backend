package job

type JobType string

const (
	JobTypeAPI JobType = "API"
)

type Job struct {
	ID             int
	OrganizationID int
	Type           JobType
	Name           string
	Enabled        bool
	CreatedAt      string
	UpdatedAt      string
}

func NewJob(organizationID int, typ JobType, enabled bool, name string) Job {
	return Job{
		OrganizationID: organizationID,
		Type:           typ,
		Enabled:        enabled,
		Name:           name,
	}
}
