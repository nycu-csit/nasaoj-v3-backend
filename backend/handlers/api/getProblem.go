package api

import (
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// GetTool
//
//	@Param		problemSHA1	query	string	true	"problem sha1"
//	@Produce	json
//	@Success	200	{object}	problemOutput
//	@Failure	500	{object}	string	"Problem loading failed"
//	@Router		/api/tool [get]
func getToolPlaceholder() {}

// GetProblem
//
//	@Param		problemSHA1	query	string	true	"problem sha1"
//	@Produce	json
//	@Success	200	{object}	problemOutput
//	@Failure	500	{object}	string	"Problem loading failed"
//	@Router		/api/problem [get]
func GetProblem(
	problems interfaces.ProblemStorage,
	isTool bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		if isTool {
			fmt.Fprintf(os.Stderr, "GET /api/tool\n")
		} else {
			fmt.Fprintf(os.Stderr, "GET /api/problem\n")
		}
		ctx := req.Context()

		problemSHA1 := req.URL.Query().Get("problemSHA1")
		if problemSHA1 == "" {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "problemSHA1 empty",
			}
		}

		var problem models.Problem
		var err error
		if isTool {
			problem, err = problems.GetTool(ctx, problemSHA1)
		} else {
			problem, err = problems.GetProblem(ctx, problemSHA1)
		}
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "get problem failed",
				OriginError: err,
			}
		}

		ret := problemOutput{
			Name:        problem.Name,
			SHA1:        problemSHA1,
			ImportedAt:  problem.ImportedAt,
			ProblemId:   problem.ProblemId,
			Description: problem.Description,
			Stages:      make(map[string]problemOutputStage),
			Tag:         problem.Tag,
			IsEnabled:   problem.IsEnabled,
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
						Name: testcase.Name,
					})
			}
			ret.Stages[stageName] = retStage
		}

		return bunrouter.JSON(w, ret)
	}
}
