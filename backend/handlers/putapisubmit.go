package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type adminSubmitInput struct {
	submitInput `json:",inline"`
	AsUser      string `json:"asUser"`
}

type submitInput struct {
	// Submit problem ID
	ProblemId int64 `json:"problemId"`
}

type submitOutput struct {
	Id          int64  `json:"id"`
	ProblemId   int64  `json:"problemId"`
	ProblemName string `json:"problemName"`

	CsId     string `json:"csId"`
	SubmitAt int64  `json:"submitAt"`
}

// PutAdminSubmit
//
//	@Param		request	body	adminSubmitInput	true	"query params"
//	@Accept		json
//	@Success	200	{object}	submitOutput
//	@Failure	400
//	@Failure	401
//	@Failure	429
//
//	@Router		/api/admin/submit [put]
func putAdminSubmitPlaceholder() {}

// PutSubmit
//
//	@Param		request	body	submitInput	true	"query params"
//	@Accept		json
//	@Success	200	{object}	submitOutput
//	@Failure	400	{object}	string
//	@Failure	401	{object}	string
//	@Failure	429	{object}	string
//
//	@Router		/api/submit [put]
func PutSubmit(
	ctx context.Context,
	db *bun.DB,
	rdb *redis.Client,
	mc *minio.Client,
	problems utils.ProblemRepository,
	testcases map[string]*template.Template,
	isAdmin bool,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "PUT /api/submit\n")

		b, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "read payload failed",
				OriginError: err,
			}
		}

		var problem models.Problem
		var resource models.UserResource
		if isAdmin {
			var input adminSubmitInput
			if err := json.Unmarshal(b, &input); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "json unmarshal failed",
					OriginError: err,
				}
			}

			var ok bool
			problem, ok = problems[input.ProblemId]
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusBadRequest,
					Message:    "problem not exist",
				}
			}

			err = db.NewSelect().Model(&resource).
				Where("csid = ?", input.AsUser).Scan(ctx)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "resource query failed or user not exist",
					OriginError: err,
				}
			}
		} else {
			var input submitInput
			if err := json.Unmarshal(b, &input); err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusBadRequest,
					Message:     "json unmarshal failed",
					OriginError: err,
				}
			}

			var ok bool
			problem, ok = problems[input.ProblemId]
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusBadRequest,
					Message:    "problem not exist",
				}
			}

			resource, ok = req.Context().Value(
				middlewares.UserResourceType("")).(models.UserResource)
			if !ok {
				return middlewares.HTTPError{
					StatusCode: http.StatusUnauthorized,
					Message:    "user resource missing",
				}
			}
		}

		// Check last submission
		fmt.Fprintf(os.Stderr, "%+v\n", problem.Timeout)
		var lastSubmit models.Submission
		err = db.NewSelect().Model(&lastSubmit).
			Where("csid = ?", resource.CsId).
			Where("problemid = ?", problem.ProblemId).
			OrderExpr("id DESC").Limit(1).Scan(ctx)
		if err == nil {
			fmt.Fprintf(os.Stderr, "lastSubmit: %+v\n", lastSubmit)
			// Check Job queue
			subJob := fmt.Sprintf("sub-job-%d", lastSubmit.Id)
			subJobCnt, err := rdb.HLen(ctx, subJob).Result()
			if err == nil && subJobCnt > 0 {
				fmt.Fprintf(os.Stderr, "subJobCnt: %+v\n", subJobCnt)
				availableAt := time.Unix(lastSubmit.SubmitAt, 0).
					Add(time.Duration(problem.Timeout) * time.Second)
				if time.Now().Unix() < availableAt.Unix() {
					w.Header().Set("Retry-After", fmt.Sprintf("%d",
						int64(availableAt.Sub(time.Now()).Seconds())))
					return middlewares.HTTPError{
						StatusCode: http.StatusTooManyRequests,
						Message:    "submit in cooldown",
					}
				}
			}
		}

		submission := models.Submission{
			ProblemName: problem.Name,
			ProblemId:   problem.ProblemId,

			CsId:     resource.CsId,
			SubmitAt: time.Now().Unix(),
			Args:     make(map[string]string),

			Score: models.UserScore{
				Score:  0,
				Stages: make(map[string]int64),
			},
		}
		fmt.Fprintf(os.Stderr, "submission: %+v\n", submission)

		for key, val := range resource.Map() {
			submission.Args[key] = val
		}

		// TODO: add tx and create jobs
		err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			err = tx.NewInsert().Model(&submission).Scan(ctx)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "submission insertion failed",
					OriginError: err,
				}
			}

			for stageName, stage := range problem.Stages {
				for _, testcase := range stage.Testcases {
					job := models.Job{
						SubId:     submission.Id,
						StageName: stageName,
						TestName:  testcase.Name,

						Status: models.StatusPending,
					}
					_, err = tx.NewInsert().Model(&job).Exec(ctx)
					if err != nil {
						return middlewares.HTTPError{
							StatusCode:  http.StatusInternalServerError,
							Message:     "job insertion failed",
							OriginError: err,
						}
					}
				}
			}
			return nil
		})
		if err != nil {
			return err
		}

		subId := submission.Id

		stageDeps := make(map[string]int64)
		for name, stage := range problem.Stages {
			_, err = rdb.HSet(ctx, "jobcnt", fmt.Sprintf("%d-%s", subId, name),
				len(stage.Testcases)).Result()
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "redis SET failed",
					OriginError: err,
				}
			}
			for _, next := range stage.Before {
				stageDeps[next] += 1
			}
		}
		fmt.Fprintf(os.Stderr, "stageDeps: %+v\n", stageDeps)

		for name := range problem.Stages {
			if _, exist := stageDeps[name]; !exist {
				err = utils.RunStage(
					ctx, db, rdb, mc, problems, testcases, submission, name)
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "RunStage failed",
						OriginError: err,
					}
				}
			} else {
				_, err = rdb.HSet(ctx, "depcnt", fmt.Sprintf("%d-%s", subId, name),
					stageDeps[name]).Result()
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "redis HSET failed",
						OriginError: err,
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, submitOutput{
			Id:          submission.Id,
			ProblemId:   submission.ProblemId,
			ProblemName: submission.ProblemName,

			CsId:     submission.CsId,
			SubmitAt: submission.SubmitAt,
		})
		return nil
	}
}
