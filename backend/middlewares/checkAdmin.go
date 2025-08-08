package middlewares

import (
	"net/http"
	"slices"

	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func CheckAdmin(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		jwtClaim, ok := req.Context().
			Value(AccessTokenType("")).(models.AccessTokenClaim)
		if !ok {
			return HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "jwt claim missing",
			}
		}

		if !slices.Contains(jwtClaim.Groups, viper.GetString("admin-role")) {
			return HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "not admin",
			}
		}
		return next(w, req)
	}
}
