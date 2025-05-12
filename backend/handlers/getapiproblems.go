package handlers

import (
	"fmt"
	"maps"
	"net/http"
	"os"
	"slices"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
)

type problemOutputStageTestcase struct {
	Name string
}

type problemOutputStage struct {
	Before    []string
	Score     int64
	Testcases []problemOutputStageTestcase
}

type problemOutput struct {
	Name      string
	ProblemId int64
	Stages    map[string]problemOutputStage
}

// GetProblems
//
// @Security ApiKeyAuth
// @Produce	json
// @Success	200	{object}	[]problemOutput
// @Failure 500 {object} string "Problem loading error"
// @Router		/api/problems [get]
func GetProblems(problems utils.ProblemRepository) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/problems\n")

		var ret []problemOutput
		for _, problem := range slices.Collect(maps.Values(problems)) {
			retProblem := problemOutput{
				Name:      problem.Name,
				ProblemId: problem.ProblemId,
				Stages:    make(map[string]problemOutputStage),
			}
			for stageName, stage := range problem.Stages {
				retStage := problemOutputStage{
					Before: stage.Before,
					Score:  stage.Score,
				}
				for _, testcase := range stage.Testcases {
					retStage.Testcases = append(retStage.Testcases,
						problemOutputStageTestcase{
							Name: testcase.Name,
						})
				}
				retProblem.Stages[stageName] = retStage
			}
			ret = append(ret, retProblem)
		}
		fmt.Fprintf(os.Stderr, "ret: %+v", ret)

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, ret)

		return nil
	}
}
