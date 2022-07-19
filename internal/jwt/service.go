package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/opsway-io/backend/internal/user"
	"github.com/pkg/errors"
)

type Service interface {
	Generate(user *user.User) (tokenString string, refreshTokenString string, err error)
	Verify(tokenString string) (valid bool, claims *Claims, err error)
	Refresh(tokenString string) (newTokenString string, newRefreshTokenString string, err error)
}

type Config struct {
	Secret           []byte        `default:"secret"`
	ExpiresIn        time.Duration `default:"1h"`
	RefreshExpiresIn time.Duration `default:"30d"`
	Issuer           string        `default:"opsway.io"`
	Audience         string        `default:"opsway.io"`
}

type ServiceImpl struct {
	Config Config
}

func NewService(conf Config) Service {
	return &ServiceImpl{
		Config: conf,
	}
}

func (s *ServiceImpl) Generate(user *user.User) (string, string, error) {
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
		Email: user.Email,
	}

	tokenString, err := s.signClaims(tokenClaims)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to generate token")
	}

	// Refresh token
	refreshTokenClaims := Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(s.Config.RefreshExpiresIn).Unix(),
			IssuedAt:  time.Now().Unix(),
			NotBefore: time.Now().Unix(),
			Issuer:    s.Config.Issuer,
			Subject:   fmt.Sprintf("%d", user.ID),
			Audience:  s.Config.Audience,
		},
	}
	refreshTokenString, err := s.signClaims(refreshTokenClaims)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to generate refresh token")
	}

	return tokenString, refreshTokenString, nil
}

func (s *ServiceImpl) Verify(tokenString string) (bool, *Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return s.Config.Secret, nil
	})
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to parse token")
	}

	if !token.Valid {
		return false, nil, nil
	}

	return true, claims, nil
}

func (s *ServiceImpl) Refresh(token string) (string, string, error) {
	return "", "", fmt.Errorf("not implemented")
}

func (s *ServiceImpl) signClaims(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.Config.Secret)
}
