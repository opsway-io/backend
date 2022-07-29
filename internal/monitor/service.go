package monitor

type Service interface{}

type ServiceImpl struct{}

func NewService() Service {
	return &ServiceImpl{}
}
