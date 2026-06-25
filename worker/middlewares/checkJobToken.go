package middlewares

import (
	"context"
	"net/http"
	"strings"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/workflows"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type ctxKey string

// JobClaimsKey is the request-context key under which CheckJobToken stores the
// verified job claims, analogous to the backend's types.Privileges key.
const JobClaimsKey ctxKey = "jobClaims"

// CheckJobToken validates the Bearer jobToken (a worker-signed JWT) and stashes
// its claims in the request context.
func CheckJobToken(
	secret []byte,
) func(bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			header := req.Header.Get("Authorization")
			token := strings.TrimPrefix(header, "Bearer ")
			if token == "" || token == header {
				return HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "missing bearer token",
				}
			}

			claims, err := workflows.VerifyJobToken(secret, token)
			if err != nil {
				return HTTPError{
					StatusCode:  http.StatusUnauthorized,
					Message:     "invalid job token",
					OriginError: err,
				}
			}

			span := trace.SpanFromContext(req.Context())
			span.SetAttributes(
				attribute.Int64("job.sub.id", claims.SubId),
				attribute.String("job.stage", claims.StageName),
				attribute.String("job.test", claims.TestName))

			ctx := context.WithValue(req.Context(), JobClaimsKey, claims)
			return next(w, req.WithContext(ctx))
		}
	}
}
