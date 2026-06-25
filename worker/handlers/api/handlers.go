package api

import (
	"net/http"

	"github.com/go-resty/resty/v2"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/worker/workflows"
)

// Handlers serves the worker-local API that jobs call to put args / scores.
// Requests are forwarded to the backend authenticated with the worker token.
type Handlers struct {
	client *resty.Client // to backend, authed with the worker token
}

func NewHandlers(client *resty.Client) *Handlers {
	return &Handlers{client: client}
}

// jobClaims pulls the claims stashed by middlewares.CheckJobToken.
func jobClaims(req *http.Request) (workflows.JobClaims, bool) {
	claims, ok := req.Context().Value(middlewares.JobClaimsKey).(workflows.JobClaims)
	return claims, ok
}

// forward mirrors the backend response (status + body) back to the job, or
// reports a gateway error when the backend is unreachable.
func (self *Handlers) forward(
	w http.ResponseWriter,
	resp *resty.Response,
	err error,
) error {
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadGateway,
			Message:     "backend request failed",
			OriginError: err,
		}
	}
	w.WriteHeader(resp.StatusCode())
	_, _ = w.Write(resp.Body())
	return nil
}
