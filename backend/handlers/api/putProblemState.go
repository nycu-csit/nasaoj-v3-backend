package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
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
//	@Failure	401	{object}	string
//	@Router		/api/admin/problem-state [put]
func PutProblemState(
	problems interfaces.ProblemStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "PUT /api/admin/problem-state\n")

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

		if err = problems.PutProblemState(
			ctx, input.ProblemSHA1, input.IsEnabled); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "problem put state failed",
				OriginError: err,
			}
		}

		fmt.Fprintf(w, "success")
		return nil
	}
}
