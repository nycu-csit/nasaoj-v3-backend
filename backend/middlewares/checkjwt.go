package middlewares

import (
	"context"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type JWTKeyType string
type UserResourceType string

func CheckJWT(
	ctx context.Context,
	db interfaces.Database,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	JWT_SECRET := os.Getenv("JWT_SECRET")

	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			jwtCookie, err := req.Cookie("jwt")
			if err != nil {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "User didn't login",
					OriginError: err,
				}
			}

			var claim models.JWTClaim
			token, err := jwt.ParseWithClaims(jwtCookie.Value, &claim,
				func(*jwt.Token) (interface{}, error) {
					return []byte(JWT_SECRET), nil
				})
			if err != nil {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "jwt cannot parse",
					OriginError: err,
				}
			}

			if !token.Valid {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "jwt invalid",
					OriginError: err,
				}
			}

			user, err := db.GetUserResource(ctx, claim.CsId)
			if err != nil {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "user not exist",
					OriginError: err,
				}
			}

			retctx := req.Context()
			retctx = context.WithValue(retctx, JWTKeyType(""), claim)
			retctx = context.WithValue(retctx, UserResourceType(""), user)
			return next(w, req.WithContext(retctx))
		}
	}
}
