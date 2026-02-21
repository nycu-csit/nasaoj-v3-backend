package api

import (
	"net/http"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

type SubmissionsOutputSubmission struct {
	Id      int64          `json:"id" yaml:"id"`
	RefSHA1 string         `json:"refSHA1" yaml:"refSHA1"`
	RefId   int64          `json:"refId" yaml:"refId"`
	Type    models.RefType `json:"type" yaml:"type"`

	OwnerId    int64 `json:"ownerId" yaml:"ownerId"`
	CreatedAt  int64 `json:"createdAt" yaml:"createdAt"`
	Score      int64 `json:"score" yaml:"score"`
	TotalScore int64 `json:"totalScore" yaml:"totalScore"`
}

type SubmissionsOutput struct {
	PageSize    int                           `json:"pageSize" yaml:"pageSize"`
	TotalRowCnt int                           `json:"totalRowCnt" yaml:"totalRowCnt"`
	Submissions []SubmissionsOutputSubmission `json:"submissions" yaml:"submissions"`
}

const pageSize int = 20

// GetAdminSubmissions
//
//	@Param		pageNum	query		int64				false	"Page number, one page is 20 submissions"	default(0)
//	@Param		ownerId	query		string				false	"Filter by OwnerId"							attribute(optional)
//	@Param		refId	query		int64				false	"Filter by RefId"							attribute(optional)
//	@Param		type	query		int					false	"Filter by Type"							attribute(optional)
//	@Success	200		{object}	SubmissionsOutput	"Submissions"
//	@Failure	400		{object}	string				"Problem Id invalid"
//	@Failure	401		{object}	string				"not admin or didn't login"
//	@Router		/api/admin/submissions [get]
func getAdminSubmissionsPlaceHolder() {}

// GetSubmissions
//
//	@Param		pageNum	query		int64				false	"Page number, one page is 20 submissions"	default(0)
//	@Param		refId	query		int64				false	"Filter by RefId"							attribute(optional)
//	@Param		type	query		int					false	"Filter by Type"							attribute(optional)
//	@Success	200		{object}	SubmissionsOutput	"Submissions"
//	@Failure	400		{object}	string				"Problem Id invalid"
//	@Failure	401		{object}	string				"didn't login"
//	@Router		/api/submissions [get]
func (self *Handlers) GetSubmissions(
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		queryValues := req.URL.Query()
		var err error

		var ownerId int64
		var isOwnerId bool = false
		if isAdmin {
			if queryValues.Has("ownerId") {
				ownerId, err = strconv.ParseInt(queryValues.Get("ownerId"), 10, 64)
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusBadRequest,
						Message:     "invalid owner id",
						OriginError: err,
					}
				}
				isOwnerId = true
			}
		} else {
			privs, ok := req.Context().
				Value(types.Privileges("")).(models.Privileges)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "privs missing",
				}
			}
			ownerId = privs.OwnerId
			isOwnerId = true
		}

		var pageNum int
		var isPageNum bool = false
		if queryValues.Has("pageNum") {
			pageNum64, err := strconv.ParseInt(
				queryValues.Get("pageNum"), 10, 32)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "page num parse failed",
					OriginError: err,
				}
			}
			pageNum = int(pageNum64)
			isPageNum = true
		}

		var refType models.RefType
		var isRefType bool = false
		if queryValues.Has("type") {
			refType64, err := strconv.ParseInt(
				queryValues.Get("type"), 10, 32)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "failed to parse 'type'",
					OriginError: err,
				}
			}
			refType = int(refType64)
			isRefType = true
		}

		var refId int64
		var isRefId bool = false
		if queryValues.Has("refId") {
			refId, err = strconv.ParseInt(
				queryValues.Get("refId"), 10, 64)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "ref id parse failed",
					OriginError: err,
				}
			}
			isRefId = true
		}

		subs, count, err := self.submissions.GetSubmissionsAndCount(ctx,
			ownerId, isOwnerId,
			pageNum, isPageNum, pageSize,
			refId, isRefId,
			refType, isRefType)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to query submissions",
				OriginError: err,
			}
		}

		submissions := []SubmissionsOutputSubmission{}
		sha1s := []string{}
		for _, sub := range subs {
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to query problem",
					OriginError: err,
				}
			}

			score, err := self.scores.GetSubmissionScore(ctx, sub.Id)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "failed to query score",
					OriginError: err,
				}
			}

			output := SubmissionsOutputSubmission{
				Id:   sub.Id,
				Type: sub.Type,

				OwnerId:   sub.OwnerId,
				CreatedAt: sub.CreatedAt,
				Score:     score.Score,
			}
			switch sub.Type {
			case models.RefTypeProblem:
				output.RefSHA1 = sub.ProblemSHA1
				output.RefId = sub.ProblemId
				sha1s = append(sha1s, output.RefSHA1)
			case models.RefTypeTool:
				output.RefSHA1 = sub.ToolSHA1
				output.RefId = sub.ToolId
			}
			submissions = append(submissions, output)
		}

		scores, err := self.problems.GetProblemScores(ctx, sha1s)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "failed to query problem scores",
				OriginError: err,
			}
		}

		for i, submission := range submissions {
			if submission.Type == models.RefTypeProblem {
				score, ok := scores[submission.RefSHA1]
				if !ok {
					return middlewares.HTTPError{
						StatusCode: http.StatusInternalServerError,
						Message:    "failed to find total score",
					}
				}
				submissions[i].TotalScore = score
			}
		}
		return bunrouter.JSON(w, SubmissionsOutput{
			PageSize:    pageSize,
			TotalRowCnt: count,
			Submissions: submissions,
		})
	}
}
