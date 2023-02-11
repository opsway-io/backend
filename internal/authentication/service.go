package authentication

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/opsway-io/backend/internal/entities"
	"github.com/pkg/errors"
)

type Service interface {
	Generate(ctx context.Context, user *entities.User) (newAccessToken string, newRefreshToken string, err error)
	Refresh(ctx context.Context, refreshToken string) (newAccessToken string, newRefreshToken string, err error)
	Verify(ctx context.Context, accessToken string) (valid bool, claims *Claims, err error)
}

type Config struct {
	Secret           string        `mapstructure:"secret"`
	ExpiresIn        time.Duration `mapstructure:"expires_in"`
	RefreshExpiresIn time.Duration `mapstructure:"refresh_expires_in"`
	Issuer           string        `mapstructure:"issuer"`
	Audience         string        `mapstructure:"audience"`
}

type ServiceImpl struct {
	Config     Config
	Repository Repository
}

func NewService(conf Config, redisClient *redis.Client) Service {
	return &ServiceImpl{
		Config: conf,
		Repository: NewRepository(
			redisClient,
		),
	}
}

func (s *ServiceImpl) Generate(ctx context.Context, user *entities.User) (accessToken string, refreshToken string, err error) {
	subject := fmt.Sprintf("%d", user.ID)

	return s.generateTokenPair(ctx, subject)
}

func (s *ServiceImpl) Refresh(ctx context.Context, refreshToken string) (newAccessToken string, newRefreshToken string, err error) {
	valid, claims, err := s.verifyRefreshToken(ctx, refreshToken)
	if err != nil {
		return "", "", errors.Wrap(err, "failed to verify token")
	}

	if !valid {
		return "", "", errors.New("invalid token")
	}

	return s.generateTokenPair(ctx, claims.Subject)
}

func (s *ServiceImpl) Verify(ctx context.Context, accessToken string) (bool, *Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(accessToken, claims, func(token *jwt.Token) (interface{}, error) {
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

func (s *ServiceImpl) verifyRefreshToken(ctx context.Context, refreshToken string) (bool, *RefreshClaims, error) {
	claims := &RefreshClaims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.Config.Secret), nil
	})
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to parse token")
	}

	if claims.Type != "refresh_token" {
		return false, nil, errors.New("invalid token type")
	}

	if !token.Valid {
		return false, nil, nil
	}

	ok, err := s.Repository.UseRefreshToken(ctx, refreshToken)
	if err != nil {
		return false, nil, errors.Wrap(err, "failed to check refresh token")
	}

	if !ok {
		return false, nil, nil
	}

	return true, claims, nil
}

func (s *ServiceImpl) generateTokenPair(ctx context.Context, subject string) (string, string, error) {
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

func (s *ServiceImpl) signClaims(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(s.Config.Secret))
}

func (s *ServiceImpl) newAccessTokenClaims(subject string) Claims {
	return Claims{
		StandardClaims: jwt.StandardClaims{
			Id:        uuid.New().String(),
			ExpiresAt: time.Now().Add(s.Config.ExpiresIn).Unix(),
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
			Id:        uuid.New().String(),
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
