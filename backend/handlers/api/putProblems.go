package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type PutProblemsInputProblem struct {
	SHA1 string `json:"sha1"`

	ProblemId   int64                   `json:"problemId"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Spec        models.ProblemSpec      `json:"spec"`
	Stages      map[string]models.Stage `json:"stages"`
	Args        map[string]string       `json:"args"`
	Tag         string                  `json:"tag"`
	PreScript   models.Content          `json:"preScript"`
	PostScript  models.Content          `json:"postScript"`
	Cooldown    int64                   `json:"cooldown"`
	Deadline    int64                   `json:"deadline"`
	ScoreType   models.ScoreTypeType    `json:"scoreType"`
}

// PutProblems
//
//	@Param		request	body	[]PutProblemsInputProblem	true	"problem"
//	@Accept		json
//	@Success	200	{object}	[]string	"SHA1s"
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

	var inputProblems []PutProblemsInputProblem
	if err := json.Unmarshal(b, &inputProblems); err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusBadRequest,
			Message:     "json unmarshal failed",
			OriginError: err,
		}
	}

	var res strings.Builder
	for _, input := range inputProblems {
		problem := models.Problem{
			ProblemId:   input.ProblemId,
			Name:        input.Name,
			Description: input.Description,
			Spec:        input.Spec,
			Stages:      input.Stages,
			Args:        input.Args,
			Tag:         input.Tag,
			PreScript:   input.PreScript,
			PostScript:  input.PostScript,
			Cooldown:    input.Cooldown,
			Deadline:    input.Deadline,
			ScoreType:   input.ScoreType,
		}
		if err := utils.CheckDeps(ctx, problem); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "dependencies check failed",
				OriginError: err,
			}
		}
		sha1, err := self.problems.InsertProblem(ctx, problem)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     fmt.Sprintf("insert problem '%s' failed", input.Name),
				OriginError: err,
			}
		}
		io.WriteString(&res, sha1+"\n")
	}

	_, err = io.WriteString(w, res.String())
	return err
}
