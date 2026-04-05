package api

import (
	"net/http"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type problemOutputStageTestcase struct {
	Name        string `json:"name"`
	Timeout     int64  `json:"timeout"`
	GracePeriod int64  `json:"gracePeriod"`
}

type problemOutputStage struct {
	Before    []string                     `json:"before"`
	Score     int64                        `json:"score"`
	Testcases []problemOutputStageTestcase `json:"testcases"`
}

type ProblemOutputSpec struct {
	Type string `json:"type"`
	Link string `json:"link"`
}

type ProblemOutput struct {
	ProblemId  int64  `json:"problemId"`
	SHA1       string `json:"sha1"`
	ImportedAt int64  `json:"importedAt"`

	Name           string                        `json:"name"`
	Description    string                        `json:"description"`
	Spec           ProblemOutputSpec             `json:"spec"`
	Stages         map[string]problemOutputStage `json:"stages"`
	Tag            string                        `json:"tag"`
	TotalScore     int64                         `json:"totalScore"`
	IsEnabled      bool                          `json:"isEnabled"`
	Deadline       int64                         `json:"deadline"`
	ScoreInputType  models.ScoreInputTypeType  `json:"scoreInputType"`
	ScoreOutputType models.ScoreOutputTypeType `json:"scoreOutputType"`
}

// GetAdminProblems
//
//	@Produce	json
//	@Success	200	{object}	[]ProblemOutput
//	@Failure	500	{object}	string	"Problem loading error"
//	@Router		/api/admin/problems [get]
func getAdminProblemsPlaceholder() {}

// GetProblems
//
//	@Produce	json
//	@Success	200	{object}	[]ProblemOutput
//	@Failure	500	{object}	string	"Problem loading error"
//	@Router		/api/problems [get]
func (self *Handlers) GetProblems(
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		var resProblems map[string]models.Problem
		var err error
		if isAdmin {
			resProblems, err = self.problems.GetAllProblems(ctx)
		} else {
			resProblems, err = self.problems.GetProblems(ctx)
		}
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to get problems",
				OriginError: err,
			}
		}

		ret := []ProblemOutput{}
		for sha1, problem := range resProblems {
			retProblem := ProblemOutput{
				Name:        problem.Name,
				SHA1:        sha1,
				ImportedAt:  problem.ImportedAt,
				ProblemId:   problem.ProblemId,
				Description: problem.Description,
				Spec: ProblemOutputSpec{
					Type: problem.Spec.Type,
					Link: problem.Spec.Link,
				},
				Stages:         make(map[string]problemOutputStage),
				Tag:            problem.Tag,
				TotalScore:     problem.TotalScore,
				IsEnabled:      problem.IsEnabled,
				Deadline:       problem.Deadline,
				ScoreInputType:  problem.ScoreInputType,
				ScoreOutputType: problem.ScoreOutputType,
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
				retProblem.Stages[stageName] = retStage
			}
			ret = append(ret, retProblem)
		}

		return bunrouter.JSON(w, ret)
	}
}
