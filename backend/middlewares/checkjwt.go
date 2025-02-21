package middlewares

import (
	"context"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type JWTKeyType string

type JWTInvalidError struct {
	Message string `json:"message"`
}

func (e JWTInvalidError) Error() string {
	return e.Message
}

func CheckJWT(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	JWT_SECRET := os.Getenv("JWT_SECRET")

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		jwtCookie, err := req.Cookie("jwt")
		if err != nil {
			return JWTInvalidError{
				Message: "jwt not exist",
			}
		}

		var claim models.JWTClaim
		token, err := jwt.ParseWithClaims(jwtCookie.Value, &claim,
			func(*jwt.Token) (interface{}, error) {
				return []byte(JWT_SECRET), nil
			})
		if err != nil {
			return JWTInvalidError{
				Message: "jwt cannot parse",
			}
		}

		if !token.Valid {
			return JWTInvalidError{
				Message: "jwt invalid",
			}
		}

		ctx = context.WithValue(ctx, JWTKeyType("jwt"), claim)
		return next(w, req.WithContext(ctx))
	}
}
