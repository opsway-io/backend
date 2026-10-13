package authentication

import (
	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	jwt.RegisteredClaims
	Type string `json:"type"`
}

type RefreshClaims struct {
	jwt.RegisteredClaims
	Type string `json:"type"`
}
