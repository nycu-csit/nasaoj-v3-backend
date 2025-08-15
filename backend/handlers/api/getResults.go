package api

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

type resultOutputJob struct {
	StageName string `json:"stageName"`
	TestName  string `json:"testName"`

	Status models.JobStatus `json:"status" enum:"260,261,262,263,460,461,462,560"`

	RetCode int   `json:"retCode"`
	ExecAt  int64 `json:"execAt"`
}

type resultOutput struct {
	Id          int64  `json:"id"`
	ProblemSHA1 string `json:"problemSHA1"`
	ProblemId   int64  `json:"problemId"`
	ProblemName string `json:"problemName"`

	CsId     string `json:"csId"`
	SubmitAt int64  `json:"submitAt"`
	Score    int64  `json:"score"`

	Jobs []resultOutputJob `json:"jobs"`
}

// GetAdminResults
//
//	@Param		subId	query		int				true	"Submission Id"
//	@Success	200		{object}	resultOutput	"Submission"
//	@Failure	400
//	@Failure	401
//	@Failure	404
//	@Router		/api/admin/results [get]
func getAdminResultsPlaceholder() {}

// GetResults
//
//	@Param		problemId	query		int				false	"Problem Id"	attribute(optional)
//	@Param		subId		query		int				false	"Submission Id"	attribute(optional)
//	@Success	200			{object}	resultOutput	"Last submission to the problem"
//	@Failure	400			{object}	string			"Problem ID invalid"
//	@Failure	401			{object}	string			"Access token missing"
//	@Failure	404			{object}	string			"No submission"
//	@Router		/api/results [get]
func GetResults(
	db interfaces.Database,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "GET /api/results\n")

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

			submission, err = db.GetSubmission(ctx, subId)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusNotFound,
					Message:     "submission query failed",
					OriginError: err,
				}
			}
		} else {
			problemIdRaw := req.URL.Query().Get("problemId")
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

				submission, err = db.GetSubmission(ctx, subId)
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusNotFound,
						Message:     "submission query failed",
						OriginError: err,
					}
				}

				if submission.CsId != privs.OwnerCsId {
					return middlewares.HTTPError{
						StatusCode:  http.StatusUnauthorized,
						Message:     "submission author mismatch",
						OriginError: err,
					}
				}
			} else if len(problemIdRaw) > 0 {
				problemId, err := strconv.ParseInt(problemIdRaw, 10, 64)
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusBadRequest,
						Message:     "invalid problem id",
						OriginError: err,
					}
				}
				submission, err = db.GetLastSubmission(ctx, privs.OwnerCsId, problemId)
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

		ret := resultOutput{
			Id:          submission.Id,
			ProblemSHA1: submission.ProblemSHA1,
			ProblemId:   submission.ProblemId,
			ProblemName: submission.ProblemName,

			CsId:     submission.CsId,
			SubmitAt: submission.SubmitAt,
			Score:    submission.Score.Score,

			Jobs: []resultOutputJob{},
		}
		for _, job := range submission.Jobs {
			ret.Jobs = append(ret.Jobs, resultOutputJob{
				StageName: job.StageName,
				TestName:  job.TestName,

				Status: job.Status,

				RetCode: job.RetCode,
				ExecAt:  job.ExecAt,
			})
		}
		bunrouter.JSON(w, ret)
		return nil
	}
}
