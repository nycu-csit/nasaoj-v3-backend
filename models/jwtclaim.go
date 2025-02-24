package models

import (
	"github.com/golang-jwt/jwt/v5"
)

// JWT fields
type JWTClaim struct {
	jwt.RegisteredClaims
	CsId   string   `json:"csid" bun:"csid,pk"`
	Groups []string `json:"groups" bun:"groups"`
	Iat    int64    `json:"iat" bun:"iat"`
	Exp    int64    `json:"exp" bun:"exp"`
}
