package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
)

// GetProblem
//
//	@Summary		Get a problem
//	@Description	Returns a single enabled problem by SHA1.
//	@Param			problemSHA1	query	string	true	"problem sha1"
//	@Produce		json
//	@Success		200	{object}	ProblemOutput
//	@Failure		400	{object}	string
//	@Failure		401	{object}	string
//	@Router			/api/problem [get]
func (self *Handlers) GetProblem(
	w http.ResponseWriter, req bunrouter.Request,
) error {
	ctx := req.Context()

	problemSHA1 := req.URL.Query().Get("problemSHA1")
	if problemSHA1 == "" {
		return middlewares.HTTPError{
			StatusCode: http.StatusBadRequest,
			Message:    "problemSHA1 empty",
		}
	}

	problem, err := self.problems.GetProblem(ctx, problemSHA1)
	if err != nil {
		return middlewares.HTTPError{
			StatusCode:  http.StatusInternalServerError,
			Message:     "get problem failed",
			OriginError: err,
		}
	}

	ret := ProblemOutput{
		Name:        problem.Name,
		SHA1:        problemSHA1,
		ImportedAt:  problem.ImportedAt,
		ProblemId:   problem.ProblemId,
		Description: problem.Description,
		Spec: ProblemOutputSpec{
			Type: problem.Spec.Type,
			Link: problem.Spec.Link,
		},
		Stages:     make(map[string]problemOutputStage),
		Tag:        problem.Tag,
		TotalScore: problem.TotalScore,
		IsEnabled:  problem.IsEnabled,
		Deadline:   problem.Deadline,
	}
	for stageName, stage := range problem.Stages {
		retStage := problemOutputStage{
			Before:    stage.Before,
			Score:     stage.Score,
			Testcases: []problemOutputStageTestcase{},
		}
		for _, testcase := range stage.Testcases {
			retStage.Testcases = append(retStage.Testcases,
				problemOutputStageTestcase{
					Name:        testcase.Name,
					Timeout:     testcase.Timeout,
					GracePeriod: testcase.GracePeriod,
				})
		}
		ret.Stages[stageName] = retStage
	}

	return bunrouter.JSON(w, ret)
}
