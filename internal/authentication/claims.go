package authentication

import (
	"github.com/golang-jwt/jwt"
)

type AccessClaims struct {
	jwt.StandardClaims
	Type string `json:"type"`
}

type RefreshClaims struct {
	jwt.StandardClaims
	Type string `json:"type"`
}
