package monitor

type Repository interface{}

type RepositoryImpl struct{}

func NewRepository() Repository {
	return &RepositoryImpl{}
}
