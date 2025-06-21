package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type RefreshTokenType string

func CheckRefreshToken(
	ctx context.Context,
	db interfaces.Database,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			refreshTokenCookie, err := req.Cookie("refresh_token")
			if err != nil {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "user didn't login",
					OriginError: err,
				}
			}

			var claim models.RefreshTokenClaim
			token, err := jwt.ParseWithClaims(refreshTokenCookie.Value, &claim,
				func(*jwt.Token) (interface{}, error) {
					return []byte(viper.GetString("REFRESH_TOKEN_SECRET")), nil
				})
			if err != nil {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "refresh token jwt cannot parse",
					OriginError: err,
				}
			}
			if !token.Valid {
				return HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "refresh token jwt invalid",
				}
			}

			// check time and refresh
			fmt.Fprintf(os.Stderr, "claim: %+v\n", claim)
			timeLeft := claim.ExpiresAt.Time.Sub(time.Now()) / time.Second
			fmt.Fprintf(os.Stderr, "%+v\n", timeLeft)
			if int64(timeLeft) < viper.GetInt64("REFRESH_TOKEN_TIMEOUT")/2 {
				if err := db.UpdateRefreshToken(ctx, claim.CsId); err != nil {
					return HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "upsert session failed",
						OriginError: err,
					}
				}
			}

			retctx := req.Context()
			retctx = context.WithValue(retctx, RefreshTokenType(""), claim)
			return next(w, req.WithContext(retctx))
		}
	}
}
