package middlewares

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func refreshAccessToken(
	ctx context.Context,
	db interfaces.Database,
	w http.ResponseWriter,
	req bunrouter.Request,
) (string, error) {
	refreshTokenClaim, ok := req.Context().
		Value(types.RefreshToken("")).(models.RefreshTokenClaim)
	if !ok {
		tracing.Logger.Ctx(ctx).
			Warn("refresh token not exist")
		return "", types.ContextNotExistError
	}

	session, err := db.GetSession(ctx, refreshTokenClaim.Id)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Warn("session not exist", zap.Error(err))
		return "", err
	}

	// Query user resource by ID (session ID is the user resource ID)
	user, err := db.GetUserResource(ctx, refreshTokenClaim.Id)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Warn("user not exist", zap.Error(err))
		return "", err
	}

	ret, err := session.ToAccessToken(user)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Warn("access token generate failed", zap.Error(err))
		return "", err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    ret,
		Path:     "/",
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return ret, nil
}

func CheckAccessToken(
	db interfaces.Database,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			ctx := req.Context()

			var accessTokenString string
			accessTokenCookie, err := req.Cookie("access_token")
			if err != nil {
				accessTokenString, err = refreshAccessToken(ctx, db, w, req)
				if err != nil {
					return HTTPError{
						StatusCode:  http.StatusUnauthorized,
						Message:     "access token refresh failed",
						OriginError: err,
					}
				}
			} else {
				accessTokenString = accessTokenCookie.Value
			}

			var claim models.AccessTokenClaim
			token, err := jwt.ParseWithClaims(accessTokenString, &claim,
				func(*jwt.Token) (interface{}, error) {
					return []byte(viper.GetString("access-token-secret")), nil
				})
			if err != nil || !token.Valid {
				accessTokenString, err = refreshAccessToken(ctx, db, w, req)
				if err != nil {
					return HTTPError{
						StatusCode:  http.StatusUnauthorized,
						Message:     "access token refresh failed",
						OriginError: err,
					}
				}
				token, err := jwt.ParseWithClaims(accessTokenString, &claim,
					func(*jwt.Token) (interface{}, error) {
						return []byte(viper.GetString("access-token-secret")), nil
					})
				if err != nil || !token.Valid {
					return HTTPError{
						StatusCode:  http.StatusUnauthorized,
						Message:     "access token jwt cannot parse or invalid",
						OriginError: err,
					}
				}
			}

			span := trace.SpanFromContext(ctx)
			span.SetAttributes(
				attribute.Int64("owner.id", claim.Privs.OwnerId))

			ctx = context.WithValue(ctx, types.AccessToken(""), claim)
			ctx = context.WithValue(ctx, types.Privileges(""), claim.Privs)
			return next(w, req.WithContext(ctx))
		}
	}
}
