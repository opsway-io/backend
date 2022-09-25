package authentication

import (
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	jwt.StandardClaims
}

type RefreshClaims struct {
	jwt.StandardClaims
	Type string `json:"type"`
}
