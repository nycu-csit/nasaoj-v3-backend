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

type resultOutputJob struct {
	StageName string `json:"stageName"`
	TestName  string `json:"testName"`

	RetCode int   `json:"retCode"`
	ExecAt  int64 `json:"execAt"`
}

type resultOutput struct {
	Id          int64  `json:"id"`
	ProblemId   int64  `json:"problemId"`
	ProblemName string `json:"problemName"`

	CsId     string `json:"csId"`
	SubmitAt int64  `json:"submitAt"`
	Score    int64  `json:"score"`
	Pending  int64  `json:"pending"`

	Jobs []resultOutputJob `json:"jobs"`
}

// GetResults
//
//	@Param		problemId	query		int				true	"Problem ID"
//	@Success	200			{object}	resultOutput	"Last submission to the problem"
//	@Failure	400			{object}	string			"Problem ID invalid"
//	@Failure	401			{object}	string			"User resource missing"
//	@Failure	404			{object}	string			"No submission"
//	@Router		/api/results [get]
func GetResults(
	ctx context.Context,
	db *bun.DB,
	rdb *redis.Client,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/results\n")

		problemId64, err := strconv.ParseInt(
			req.URL.Query().Get("problemId"), 10, 32)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return fmt.Errorf("invalid problem id: %w\n", err)
		}
		problemId := int(problemId64)

		resource, ok := req.Context().
			Value(middlewares.UserResourceType("")).(models.UserResource)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return fmt.Errorf("user resource missing\n")
		}

		var submission models.Submission
		err = db.NewSelect().Model(&submission).
			Where("csid = ?", resource.CsId).
			Where("problemid = ?", problemId).
			Relation("Jobs").OrderExpr("id DESC").Limit(1).Scan(ctx)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return fmt.Errorf("submission query failed, %s\n", err)
		}

		jobcnt, err := rdb.HLen(ctx,
			fmt.Sprintf("sub-job-%s", submission.Id)).Result()
		if err != nil {
			jobcnt = 0
		}

		w.WriteHeader(http.StatusOK)
		ret := resultOutput{
			Id:          submission.Id,
			ProblemId:   submission.ProblemId,
			ProblemName: submission.ProblemName,

			CsId:     submission.CsId,
			SubmitAt: submission.SubmitAt,
			Score:    submission.Score,
			Pending:  jobcnt,
		}
		for _, job := range submission.Jobs {
			ret.Jobs = append(ret.Jobs, resultOutputJob{
				StageName: job.StageName,
				TestName:  job.TestName,

				RetCode: job.RetCode,
				ExecAt:  job.ExecAt,
			})
		}
		bunrouter.JSON(w, ret)
		return nil
	}
}
