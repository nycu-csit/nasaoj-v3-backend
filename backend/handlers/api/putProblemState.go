package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type PutProblemStateInput struct {
	ProblemSHA1 string `json:"problemSHA1"`
	IsEnabled   bool   `json:"isEnabled"`
}

// PutProblemState
//
//	@Param		request	body	PutProblemStateInput	true	"problem state"
//	@Accept		json
//	@Success	200	{object}	string	"success"
//	@Failure	400	{object}	string
//	@Failure	401	{object}	string
//	@Router		/api/admin/problem/state [put]
func PutProblemState(
	problems interfaces.ProblemStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		b, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "read payload failed",
				OriginError: err,
			}
		}

		var input PutProblemStateInput
		if err := json.Unmarshal(b, &input); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "json unmarshal failed",
				OriginError: err,
			}
		}

		span := trace.SpanFromContext(ctx)
		span.SetAttributes(
			attribute.String("problem.sha1", input.ProblemSHA1))

		if err = problems.UpdateProblemState(
			ctx, input.ProblemSHA1, input.IsEnabled); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "problem put state failed",
				OriginError: err,
			}
		}

		return utils.Success(w)
	}
}
