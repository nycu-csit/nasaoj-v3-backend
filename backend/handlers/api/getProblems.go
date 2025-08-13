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

type problemOutputStageTestcase struct {
	Name string `json:"name"`
}

type problemOutputStage struct {
	Before    []string                     `json:"before"`
	Score     int64                        `json:"score"`
	Testcases []problemOutputStageTestcase `json:"testcases"`
}

type problemOutput struct {
	ProblemId int64  `json:"problemId"`
	SHA1      string `json:"sha1"`

	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Stages      map[string]problemOutputStage `json:"stages"`
	Tag         string                        `json:"tag"`
	IsEnabled   bool                          `json:"isEnabled"`
}

// GetAdminTools
//
//	@Produce	json
//	@Success	200	{object}	[]problemOutput
//	@Failure	500	{object}	string	"Tool loading error"
//	@Router		/api/admin/tools [get]
func getAdminToolsPlaceholder() {}

// GetTools
//
//	@Produce	json
//	@Success	200	{object}	[]problemOutput
//	@Failure	500	{object}	string	"Tool loading error"
//	@Router		/api/tools [get]
func getToolsPlaceholder() {}

// GetAdminProblems
//
//	@Produce	json
//	@Success	200	{object}	[]problemOutput
//	@Failure	500	{object}	string	"Problem loading error"
//	@Router		/api/admin/problems [get]
func getAdminProblemsPlaceholder() {}

// GetProblems
//
//	@Produce	json
//	@Success	200	{object}	[]problemOutput
//	@Failure	500	{object}	string	"Problem loading error"
//	@Router		/api/problems [get]
func GetProblems(
	db interfaces.Database,
	isTool bool,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		if isAdmin {
			if isTool {
				fmt.Fprintf(os.Stderr, "GET /api/admin/tools\n")
			} else {
				fmt.Fprintf(os.Stderr, "GET /api/admin/problems\n")
			}
		} else {
			if isTool {
				fmt.Fprintf(os.Stderr, "GET /api/tools\n")
			} else {
				fmt.Fprintf(os.Stderr, "GET /api/problems\n")
			}
		}
		ctx := req.Context()

		var problems map[string]models.Problem
		var err error
		if isAdmin {
			if isTool {
				problems, err = db.GetAllTools(ctx)
			} else {
				problems, err = db.GetAllProblems(ctx)
			}
		} else {
			if isTool {
				problems, err = db.GetTools(ctx)
			} else {
				problems, err = db.GetProblems(ctx)
			}
		}
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "get problems failed",
				OriginError: err,
			}
		}

		ret := []problemOutput{}
		for sha1, problem := range problems {
			retProblem := problemOutput{
				Name:        problem.Name,
				SHA1:        sha1,
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
				retProblem.Stages[stageName] = retStage
			}
			ret = append(ret, retProblem)
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, ret)
		return nil
	}
}
