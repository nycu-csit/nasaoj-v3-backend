package models

import (
	"github.com/golang-jwt/jwt/v5"
)

type JWTClaim struct {
	jwt.RegisteredClaims
	CsId   string   `json:"csid"`
	Groups []string `json:"groups"`
	Iat    int64    `json:"iat"`
	Exp    int64    `json:"exp"`
}
