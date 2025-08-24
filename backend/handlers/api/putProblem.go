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
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// PutProblems
//
//	@Param		request	body	map[string]models.Problem	true	"problem"
//	@Accept		json
//	@Success	200	{object}	string	"success"
//	@Failure	401	{object}	string
//	@Router		/api/admin/problems [put]
func PutProblems(
	problems interfaces.ProblemStorage,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "PUT /api/admin/problems\n")

		b, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "read payload failed",
				OriginError: err,
			}
		}

		// fmt.Fprintf(os.Stderr, "%+v\n", string(b[:]))
		inputProblems := make(map[string]models.Problem)
		// fmt.Fprintf(os.Stderr, "%+v\n", inputProblems)
		if err := json.Unmarshal(b, &inputProblems); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "json unmarshal failed",
				OriginError: err,
			}
		}

		for sha1, problem := range inputProblems {
			if err := utils.CheckDeps(ctx, problem); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "dependencies check failed",
					OriginError: err,
				}
			}
			// fmt.Fprintf(os.Stderr, "[INFO] Insert problem '%s' with sha1 '%s'\n",
			// 	problem.Name, sha1)
			if err := problems.InsertProblem(ctx, problem); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     fmt.Sprintf("insert problem '%s' failed", sha1),
					OriginError: err,
				}
			}
		}

		fmt.Fprintf(w, "success")
		return nil
	}
}
