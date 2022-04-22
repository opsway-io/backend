package job

type Cache interface {
	Set(job Job) (err error)
	Delete(jobID int) (err error)
	Get(jobID int) (job Job, ok bool, err error)
}

func NewCache() (Cache, error) {
	// TODO: implement

	return nil, nil
}
