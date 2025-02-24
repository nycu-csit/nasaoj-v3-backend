package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type JWTKeyType string
type UserResourceType string

type JWTInvalidError struct {
	Message string `json:"message"`
}

func (e JWTInvalidError) Error() string {
	return e.Message
}

func CheckJWT(
	ctx context.Context,
	db *bun.DB,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	JWT_SECRET := os.Getenv("JWT_SECRET")

	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
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

			var user models.UserResource
			err = db.NewSelect().Model(&user).Where("csid = ?", claim.CsId).Scan(ctx)
			if err != nil {
				return fmt.Errorf("UserResource for csid: %s not exist\n", claim.CsId)
			}

			retctx := req.Context()
			retctx = context.WithValue(retctx, JWTKeyType(""), claim)
			retctx = context.WithValue(retctx, UserResourceType(""), user)
			return next(w, req.WithContext(retctx))
		}
	}
}
