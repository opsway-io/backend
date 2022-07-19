package jwt

import (
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	jwt.StandardClaims
	Email string `json:"email"`
}

type RefreshClaims struct {
	jwt.StandardClaims
	Type string `json:"type"`
}
