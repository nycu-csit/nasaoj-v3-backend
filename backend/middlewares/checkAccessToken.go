package middlewares

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type AccessTokenType string

func CheckAccessToken(
	ctx context.Context,
	db interfaces.Database,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			refreshTokenClaim, ok := req.Context().
				Value(RefreshTokenType("")).(models.RefreshTokenClaim)
			if !ok {
				return HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "refresh token not exist",
				}
			}

			var accessTokenString string
			accessTokenCookie, err := req.Cookie("access_token")
			if err != nil {
				session, err := db.GetSession(ctx, refreshTokenClaim.CsId)
				if err != nil {
					return HTTPError{
						StatusCode:  http.StatusUnauthorized,
						Message:     "session not exist",
						OriginError: err,
					}
				}

				user, err := db.GetUserResource(ctx, refreshTokenClaim.CsId)
				if err != nil {
					return HTTPError{
						StatusCode:  http.StatusUnauthorized,
						Message:     "user not exist",
						OriginError: err,
					}
				}

				accessTokenString, err = session.ToAccessToken(user)
				if err != nil {
					return HTTPError{
						StatusCode:  http.StatusUnauthorized,
						Message:     "access token generate failed",
						OriginError: err,
					}
				}

				http.SetCookie(w, &http.Cookie{
					Name:     "access_token",
					Value:    accessTokenString,
					Path:     "/",
					Secure:   false,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			} else {
				accessTokenString = accessTokenCookie.Value
			}

			var claim models.AccessTokenClaim
			token, err := jwt.ParseWithClaims(accessTokenString, &claim,
				func(*jwt.Token) (interface{}, error) {
					return []byte(viper.GetString("ACCESS_TOKEN_SECRET")), nil
				})
			if err != nil {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "access token jwt cannot parse",
					OriginError: err,
				}
			}

			if !token.Valid {
				return HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "access token jwt invalid",
				}
			}

			retctx := req.Context()
			retctx = context.WithValue(retctx, AccessTokenType(""), claim)
			return next(w, req.WithContext(retctx))
		}
	}
}
