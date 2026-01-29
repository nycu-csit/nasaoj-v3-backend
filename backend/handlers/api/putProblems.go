package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
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
func (self *Handlers) PutProblems(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	b, err := io.ReadAll(req.Body)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "read payload failed",
			OriginError: err,
		}
	}

	inputProblems := make(map[string]models.Problem)
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
		if err := self.problems.InsertProblem(ctx, problem); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     fmt.Sprintf("insert problem '%s' failed", sha1),
				OriginError: err,
			}
		}
	}

	return utils.Success(w)
}
