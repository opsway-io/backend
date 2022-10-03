package storage

import "io"

type Service interface {
	GetAvatarURL(userID uint) (string, error)
	UploadAvatar(data io.Reader, userID uint) ([]byte, error)
}

type ServiceImpl struct{}

func NewService(repository Repository) Service {
	return &ServiceImpl{}
}

func (s *ServiceImpl) GetAvatarURL(userID uint) (string, error) {
	return "", nil
}

func (s *ServiceImpl) UploadAvatar(data io.Reader, userID uint) ([]byte, error) {
	return nil, nil
}
