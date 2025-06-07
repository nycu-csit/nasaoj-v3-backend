package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type submissionsOutputSubmission struct {
	Id          int64  `json:"id"`
	ProblemId   int64  `json:"problemId"`
	ProblemName string `json:"problemName"`

	CsId     string `json:"csId"`
	SubmitAt int64  `json:"submitAt"`
	Score    int64  `json:"score"`
}

type submissionsOutput struct {
	PageSize    int                           `json:"pageSize"`
	TotalRowCnt int                           `json:"totalRowCnt"`
	Submissions []submissionsOutputSubmission `json:"submissions"`
}

const pageSize int = 20

// GetAdminSubmissions
//
//	@Param		pageNum		query		int64				false	"Page number, one page is 20 submissions"	default(0)
//	@Param		csId		query		string				false	"Filter by CsId"							attribute(optional)
//	@Param		problemId	query		int64				false	"Filter by Problem Id"						attribute(optional)
//
//	@Success	200			{object}	submissionsOutput	"Submissions"
//	@Failure	400			{object}	string				"Problem Id invalid"
//	@Failure	401			{object}	string				"not admin or didn't login"
//	@Router		/api/admin/submissions [get]
func getAdminSubmissionsPlaceHolder() {}

// GetSubmissions
//
//	@Param		pageNum		query		int64				false	"Page number, one page is 20 submissions"	default(0)
//	@Param		problemId	query		int64				false	"Filter by Problem Id"						attribute(optional)
//	@Success	200			{object}	submissionsOutput	"Submissions"
//	@Failure	400			{object}	string				"Problem Id invalid"
//	@Failure	401			{object}	string				"didn't login"
//	@Router		/api/submissions [get]
func GetSubmissions(
	ctx context.Context,
	db interfaces.Database,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/submissions\n")
		queryValues := req.URL.Query()
		var err error

		var csId string
		var isCsId bool = false
		if isAdmin {
			if queryValues.Has("csId") {
				csId = queryValues.Get("csId")
				isCsId = true
			}
		} else {
			resource, ok := req.Context().
				Value(middlewares.UserResourceType("")).(models.UserResource)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "user resource missing",
				}
			}
			csId = resource.CsId
			isCsId = true
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

		var problemId int64
		var isProblemId bool = false
		if queryValues.Has("problemId") {
			problemId, err = strconv.ParseInt(
				queryValues.Get("problemId"), 10, 64)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "problem id parse failed",
					OriginError: err,
				}
			}
			isProblemId = true
		}

		subs, count, err := db.GetSubmissionsAndCount(ctx,
			csId, isCsId,
			pageNum, isPageNum, pageSize,
			problemId, isProblemId)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "db query failed",
				OriginError: err,
			}
		}

		submissions := []submissionsOutputSubmission{}
		for _, sub := range subs {
			output := submissionsOutputSubmission{
				Id:          sub.Id,
				ProblemId:   sub.ProblemId,
				ProblemName: sub.ProblemName,

				CsId:     sub.CsId,
				SubmitAt: sub.SubmitAt,
				Score:    sub.Score.Score,
			}
			submissions = append(submissions, output)
		}
		bunrouter.JSON(w, submissionsOutput{
			PageSize:    pageSize,
			TotalRowCnt: count,
			Submissions: submissions,
		})
		return nil
	}
}
