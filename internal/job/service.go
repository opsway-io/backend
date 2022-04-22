package job

import (
	"errors"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var (
	ErrJobNotFound    = errors.New("job not found")
	ErrJobNameInvalid = errors.New("job name invalid")
)

type Service interface {
	Create(job Job) (err error)
	Delete(jobID int) (err error)
	SetEnabled(jobID int, enabled bool) (err error)
	SetName(jobID int, name string) (err error)
	List(organizationID int) (jobs []Job, err error)
}

func NewService(db *gorm.DB, redisClient *redis.Client) (Service, error) {
	// TODO: implement

	return nil, nil
}
