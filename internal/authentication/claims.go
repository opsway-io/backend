package authentication

import (
	"github.com/golang-jwt/jwt"
)

type Claims struct {
	jwt.StandardClaims
	Email  string `json:"email"`
	TeamID int    `json:"teamId"`
}

type RefreshClaims struct {
	jwt.StandardClaims
	Type string `json:"type"`
}
