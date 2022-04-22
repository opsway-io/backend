package job

import "gorm.io/gorm"

type Repository interface {
	Create(job Job) (err error)
	Delete(jobID int) (err error)
	SetEnabled(jobID int, enabled bool) (err error)
	SetName(jobID int, name string) (err error)
	List(organizationID int) (jobs []Job, err error)
}

func NewRepository(db *gorm.DB) (Repository, error) {
	// TODO: implement

	return nil, nil
}
