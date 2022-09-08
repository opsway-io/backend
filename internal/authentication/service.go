package authentication

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/pkg/errors"
)

type Service interface {
	Generate(user *entities.User) (tokenString string, err error)
	Verify(tokenString string) (valid bool, claims *Claims, err error)
}

type Config struct {
	Secret           string        `mapstructure:"secret"`
	ExpiresIn        time.Duration `mapstructure:"expires_in"`
	RefreshExpiresIn time.Duration `mapstructure:"refresh_expires_in"`
	Issuer           string        `mapstructure:"issuer"`
	Audience         string        `mapstructure:"audience"`
}

type ServiceImpl struct {
	Config Config
}

func NewService(conf Config) Service {
	return &ServiceImpl{
		Config: conf,
	}
}

func (s *ServiceImpl) Generate(user *entities.User) (string, error) {
	// Token
	tokenClaims := Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(s.Config.ExpiresIn).Unix(),
			IssuedAt:  time.Now().Unix(),
			NotBefore: time.Now().Unix(),
			Issuer:    s.Config.Issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			Audience:  s.Config.Audience,
		},
		Email:  user.Email,
		TeamID: 42, // TODO: get team id from user
	}

	tokenString, err := s.signClaims(tokenClaims)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate token")
	}

	return tokenString, nil
}

func (s *ServiceImpl) Verify(tokenString string) (bool, *Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.Config.Secret), nil
	})
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to parse token")
	}

	if !token.Valid {
		return false, nil, nil
	}

	return true, claims, nil
}

func (s *ServiceImpl) signClaims(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(s.Config.Secret))
}
