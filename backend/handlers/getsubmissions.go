package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
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
//	@Success	200			{object}	submissionsOutput	"All submission within the problem"
//	@Failure	400			{object}	string				"Problem Id invalid"
//	@Failure	401			{object}	string				"not admin or didn't login"
//	@Router		/api/admin/submissions [get]
func GetSubmissions(
	ctx context.Context,
	db *bun.DB,
	rdb *redis.Client,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/admin/submissions\n")
		queryValues := req.URL.Query()

		dbQuery := db.NewSelect().Order("id DESC").Limit(pageSize)

		if queryValues.Has("pageNum") {
			pageNum, err := strconv.ParseInt(
				queryValues.Get("pageNum"), 10, 32)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "page num parse failed",
					OriginError: err,
				}
			}

			dbQuery = dbQuery.Offset(int(pageNum) * pageSize)
		}

		if queryValues.Has("csId") {
			csId := queryValues.Get("csId")
			dbQuery = dbQuery.Where("csid = ?", csId)
		}

		if queryValues.Has("problemId") {
			problemId, err := strconv.ParseInt(
				queryValues.Get("problemId"), 10, 64)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "problem id parse failed",
					OriginError: err,
				}
			}

			dbQuery = dbQuery.Where("problemid = ?", problemId)
		}

		var subs []models.Submission
		count, err := dbQuery.Model(&subs).ScanAndCount(ctx)
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
