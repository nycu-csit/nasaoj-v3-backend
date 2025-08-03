package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type RefreshTokenType string

func CheckRefreshToken(
	db interfaces.Database,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			ctx := req.Context()

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
			timeLeft := claim.ExpiresAt.Time.Sub(time.Now()) / time.Second
			if int64(timeLeft) < viper.GetInt64("REFRESH_TOKEN_TIMEOUT")/2 {
				session, err := db.UpdateRefreshToken(ctx, claim.CsId)
				if err != nil {
					return HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "upsert session failed",
						OriginError: err,
					}
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "refresh_token",
					Value:    session.RefreshToken,
					Path:     "/",
					Secure:   false,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			ctx = context.WithValue(ctx, RefreshTokenType(""), claim)
			return next(w, req.WithContext(ctx))
		}
	}
}
