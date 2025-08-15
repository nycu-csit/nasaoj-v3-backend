package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

func CheckAuth(
	db interfaces.Database,
	tokens interfaces.TokenStorage,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			authHeader := strings.Split(req.Header.Get("Authorization"), " ")
			// fmt.Fprintf(os.Stderr, "%+v\n", authHeader)
			if len(authHeader) != 2 || authHeader[0] != "Bearer" {
				return CheckRefreshToken(db)(
					CheckAccessToken(db)(next),
				)(w, req)
			}

			// Auth with access token
			ctx := req.Context()
			privs, err := tokens.GetPrivs(ctx, authHeader[1])
			if err != nil {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "access token not found or expired",
					OriginError: err,
				}
			}
			// fmt.Fprintf(os.Stderr, "[INFO] access token accept, %v\n", privs)
			ctx = context.WithValue(ctx, types.Privileges(""), privs)
			return next(w, req.WithContext(ctx))
		}
	}
}
