package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

func CheckAuth(
	db interfaces.Database,
	tokens interfaces.TokenStorage,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			authHeader := strings.Split(req.Header.Get("Authorization"), " ")
			if len(authHeader) != 2 || authHeader[0] != "Bearer" {
				return CheckRefreshToken(db)(
					CheckAccessToken(db)(next),
				)(w, req)
			}

			// Auth with static bearer token
			ctx := req.Context()
			token, err := tokens.GetToken(ctx, authHeader[1])
			if err != nil || !token.IsEnabled {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "access token not found or expired",
					OriginError: err,
				}
			}

			privs := token.Privs
			if token.UserId != 0 {
				// User token: resolve privileges and namespace membership
				// live from the user, so revocation, promotion/demotion, and
				// namespace membership changes take effect immediately
				// without needing to reissue the token.
				user, err := db.GetUserResource(ctx, token.UserId)
				if err != nil {
					return HTTPError{
						StatusCode:  http.StatusUnauthorized,
						Message:     "token owner not found",
						OriginError: err,
					}
				}
				if user.IsAdmin {
					privs = models.NewAdminPrivs(user.Id)
				} else {
					privs = models.NewUserPrivs(user.Id)
				}
				ctx = context.WithValue(ctx, types.AccessToken(""), models.AccessTokenClaim{
					AccessTokenClaimFields: models.AccessTokenClaimFields{
						Resource: user,
						Privs:    privs,
					},
				})
			}

			ctx = context.WithValue(ctx, types.Privileges(""), privs)
			return next(w, req.WithContext(ctx))
		}
	}
}
