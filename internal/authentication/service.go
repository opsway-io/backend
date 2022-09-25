package authentication

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/pkg/errors"
)

type Service interface {
	Generate(user *entities.User) (newAccessToken string, newRefreshToken string, err error)
	Refresh(refreshToken string) (newAccessToken string, newRefreshToken string, err error)
	VerifyAccessToken(tokenString string) (valid bool, claims *Claims, err error)
	VerifyRefreshToken(tokenString string) (valid bool, claims *RefreshClaims, err error)
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

func (s *ServiceImpl) Generate(user *entities.User) (accessToken string, refreshToken string, err error) {
	subject := fmt.Sprintf("%d", user.ID)

	return s.generateTokenPair(subject)
}

func (s *ServiceImpl) Refresh(refreshToken string) (newAccessToken string, newRefreshToken string, err error) {
	valid, claims, err := s.VerifyRefreshToken(refreshToken)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to verify token")
	}

	if !valid {
		return "", "", errors.New("invalid token")
	}

	tokenClaims := s.newAccessTokenClaims(claims.Subject)

	return s.generateTokenPair(tokenClaims.Subject)
}

func (s *ServiceImpl) VerifyAccessToken(tokenString string) (bool, *Claims, error) {
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

func (s *ServiceImpl) VerifyRefreshToken(tokenString string) (bool, *RefreshClaims, error) {
	claims := &RefreshClaims{}
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

func (s *ServiceImpl) newAccessTokenClaims(subject string) Claims {
	return Claims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Second * 10).Unix(),
			IssuedAt:  time.Now().Unix(),
			NotBefore: time.Now().Unix(),
			Issuer:    s.Config.Issuer,
			Subject:   subject,
			Audience:  s.Config.Audience,
		},
	}
}

func (s *ServiceImpl) newRefreshTokenClaims(subject string) RefreshClaims {
	return RefreshClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(s.Config.RefreshExpiresIn).Unix(),
			IssuedAt:  time.Now().Unix(),
			NotBefore: time.Now().Unix(),
			Issuer:    s.Config.Issuer,
			Subject:   subject,
			Audience:  s.Config.Audience,
		},
		Type: "refresh_token",
	}
}

func (s *ServiceImpl) signClaims(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(s.Config.Secret))
}

func (s *ServiceImpl) generateTokenPair(subject string) (string, string, error) {
	accessTokenClaims := s.newAccessTokenClaims(subject)
	accessTokenString, err := s.signClaims(accessTokenClaims)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to generate token")
	}

	refreshTokenClaims := s.newRefreshTokenClaims(subject)
	refreshTokenString, err := s.signClaims(refreshTokenClaims)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to generate token")
	}

	return accessTokenString, refreshTokenString, nil
}
