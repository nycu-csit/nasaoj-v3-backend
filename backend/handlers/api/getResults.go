package api

import (
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

type ResultOutputJob struct {
	StageName string `json:"stageName"`
	TestName  string `json:"testName"`

	Status models.JobStatus `json:"status" enum:"260,261,262,263,460,461,462,560"`

	RetCode int   `json:"retCode"`
	ExecAt  int64 `json:"execAt"`
}

type ResultOutput struct {
	Id      int64          `json:"id"`
	RefSHA1 string         `json:"refSHA1"`
	RefId   int64          `json:"refId"`
	Type    models.RefType `json:"type"`

	OwnerId   int64             `json:"ownerId"`
	InputArgs map[string]string `json:"inputArgs"`

	CreatedAt  int64 `json:"createdAt"`
	Score      int64 `json:"score"`
	TotalScore int64 `json:"totalScore"`

	Jobs []ResultOutputJob `json:"jobs"`
}

// GetAdminResults
//
//	@Param		subId	query		int				true	"Submission Id"
//	@Success	200		{object}	ResultOutput	"Submission"
//	@Failure	400
//	@Failure	401
//	@Failure	404
//	@Router		/api/admin/results [get]
func getAdminResultsPlaceholder() {}

// GetResults
//
//	@Param		refId	query		int				false	"Ref Id"			attribute(optional)
//	@Param		type	query		models.RefType	false	"Submission Type"	attribute(optional)
//	@Param		subId	query		int				false	"Submission Id"		attribute(optional)
//	@Success	200		{object}	ResultOutput	"Last submission to the ref / tool"
//	@Failure	400
//	@Failure	401
//	@Failure	404
//	@Router		/api/results [get]
func (self *Handlers) GetResults(
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		var submission models.Submission
		if isAdmin {
			subId, err := strconv.ParseInt(
				req.URL.Query().Get("subId"), 10, 64)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "invalid submission id",
					OriginError: err,
				}
			}

			submission, err = self.submissions.GetSubmission(ctx, subId)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusNotFound,
					Message:     "submission query failed",
					OriginError: err,
				}
			}
		} else {
			refIdRaw := req.URL.Query().Get("refId")
			typeRaw := req.URL.Query().Get("type")
			subIdRaw := req.URL.Query().Get("subId")

			privs, ok := req.Context().
				Value(types.Privileges("")).(models.Privileges)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "access token missing",
				}
			}

			if len(subIdRaw) > 0 {
				subId, err := strconv.ParseInt(subIdRaw, 10, 64)
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusBadRequest,
						Message:     "invalid submission id",
						OriginError: err,
					}
				}

				submission, err = self.submissions.GetSubmission(ctx, subId)
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusNotFound,
						Message:     "submission query failed",
						OriginError: err,
					}
				}

				if submission.OwnerId != privs.OwnerId {
					return middlewares.HTTPError{
						StatusCode:  http.StatusForbidden,
						Message:     "submission author mismatch",
						OriginError: err,
					}
				}
			} else if len(refIdRaw) > 0 && len(typeRaw) > 0 {
				refId, err := strconv.ParseInt(refIdRaw, 10, 64)
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusBadRequest,
						Message:     "invalid ref id",
						OriginError: err,
					}
				}
				subType, err := strconv.ParseInt(typeRaw, 10, 32)
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusBadRequest,
						Message:     "invalid submission type",
						OriginError: err,
					}
				}
				submission, err = self.submissions.GetLastSubmission(
					ctx, privs.OwnerId, refId, int(subType))
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusNotFound,
						Message:     "submission query failed or no previous submission",
						OriginError: err,
					}
				}
			} else {
				return middlewares.HTTPError{
					StatusCode: http.StatusBadRequest,
					Message:    "invalid query param",
				}
			}
		}

		ret := ResultOutput{
			Id:   submission.Id,
			Type: submission.Type,

			OwnerId:   submission.OwnerId,
			InputArgs: submission.InputArgs,

			CreatedAt: submission.CreatedAt,

			Jobs: []ResultOutputJob{},
		}

		switch ret.Type {
		case models.RefTypeProblem:
			ret.RefSHA1 = submission.ProblemSHA1
			ret.RefId = submission.ProblemId
			problem, err := self.problems.GetProblem(ctx, submission.ProblemSHA1)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to get problem",
					OriginError: err,
				}
			}
			ret.TotalScore = problem.TotalScore

			score, err := self.scores.GetSubmissionScore(ctx, submission.Id)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to get submission score",
					OriginError: err,
				}
			}
			ret.Score = score.Score
		case models.RefTypeTool:
			ret.RefSHA1 = submission.ToolSHA1
			ret.RefId = submission.ToolId
		}
		for _, job := range submission.Jobs {
			ret.Jobs = append(ret.Jobs, ResultOutputJob{
				StageName: job.StageName,
				TestName:  job.TestName,

				Status: job.Status,

				RetCode: job.RetCode,
				ExecAt:  job.ExecAt,
			})
		}
		return bunrouter.JSON(w, ret)
	}
}
