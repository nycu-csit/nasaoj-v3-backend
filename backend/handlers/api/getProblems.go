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

	Name            string                        `json:"name"`
	Description     string                        `json:"description"`
	Spec            ProblemOutputSpec             `json:"spec"`
	Stages          map[string]problemOutputStage `json:"stages"`
	Tag             string                        `json:"tag"`
	TotalScore      int64                         `json:"totalScore"`
	IsEnabled       bool                          `json:"isEnabled"`
	Deadline        int64                         `json:"deadline"`
	ScoreInputType  models.ScoreInputTypeType     `json:"scoreInputType"`
	ScoreOutputType models.ScoreOutputTypeType    `json:"scoreOutputType"`
}

func toProblemOutput(sha1 string, problem models.Problem) ProblemOutput {
	ret := ProblemOutput{
		Name:        problem.Name,
		SHA1:        sha1,
		ImportedAt:  problem.ImportedAt,
		ProblemId:   problem.ProblemId,
		Description: problem.Description,
		Spec: ProblemOutputSpec{
			Type: problem.Spec.Type,
			Link: problem.Spec.Link,
		},
		Stages:          make(map[string]problemOutputStage),
		Tag:             problem.Tag,
		TotalScore:      problem.TotalScore,
		IsEnabled:       problem.IsEnabled,
		Deadline:        problem.Deadline,
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
		ret.Stages[stageName] = retStage
	}
	return ret
}

// ProblemDetailOutput is the full problem definition - every field,
// including testcase file names/SHA1/payload - as opposed to ProblemOutput's
// trimmed summary. Admin only: it can expose script/testcase content TAs
// may not want visible to students.
type ProblemDetailOutput struct {
	models.Problem `json:",inline" yaml:",inline"`
	SHA1           string `json:"sha1" yaml:"sha1"`
}

// GetAdminProblems
//
//	@Summary		List all problems (admin)
//	@Description	Returns all problems, including disabled ones. Pass
//	@Description	detail=true for the full problem definition (every field,
//	@Description	including testcase file hashes) instead of the summary.
//	@Description	Admin only.
//	@Produce		json
//	@Param			detail	query		bool	false	"return the full problem definition"
//	@Success		200		{object}	[]ProblemOutput
//	@Failure		500		{object}	string	"Problem loading error"
//	@Router			/api/admin/problems [get]
func getAdminProblemsPlaceholder() {}

// GetProblems
//
//	@Summary		List problems
//	@Description	Returns the enabled problems visible to the caller.
//	@Produce		json
//	@Success		200	{object}	[]ProblemOutput
//	@Failure		500	{object}	string	"Problem loading error"
//	@Router			/api/problems [get]
func (self *Handlers) GetProblems(
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		// Scope the listing to the requested namespace(s) in storage, so other
		// namespaces' problems are never loaded. Storage reads nil as "every
		// namespace", so fail closed if none were resolved.
		namespaceIds := middlewares.NamespaceIdsFromContext(ctx)
		if len(namespaceIds) == 0 {
			return bunrouter.JSON(w, []ProblemOutput{})
		}

		var resProblems map[string]models.Problem
		var err error
		if isAdmin {
			resProblems, err = self.problems.GetAllProblems(ctx, namespaceIds)
		} else {
			resProblems, err = self.problems.GetProblems(ctx, namespaceIds)
		}
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to get problems",
				OriginError: err,
			}
		}

		detail := isAdmin && req.URL.Query().Get("detail") == "true"

		if detail {
			ret := []ProblemDetailOutput{}
			for sha1, problem := range resProblems {
				ret = append(ret, ProblemDetailOutput{Problem: problem, SHA1: sha1})
			}
			return bunrouter.JSON(w, ret)
		}

		ret := []ProblemOutput{}
		for sha1, problem := range resProblems {
			ret = append(ret, toProblemOutput(sha1, problem))
		}

		return bunrouter.JSON(w, ret)
	}
}
