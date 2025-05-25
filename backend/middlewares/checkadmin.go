package middlewares

import (
	"context"
	"net/http"
	"slices"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/env"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func CheckAdmin(
	ctx context.Context,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			jwtClaim, ok := req.Context().
				Value(JWTKeyType("")).(models.JWTClaim)
			if !ok {
				return HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "jwt claim missing",
				}
			}

			if !slices.Contains(jwtClaim.Groups, env.ADMIN_ROLE) {
				return HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "not admin",
				}
			}
			return next(w, req)
		}
	}
}
