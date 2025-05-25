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

// PutSubmit
//
//	@Security	ApiKeyAuth
//	@Param		request	body	submitInput	true	"query params"
//	@Accept		json
//	@Success	200	{object}	submitOutput
//	@Failure	400	{object}	string
//	@Failure	401	{object}	string
//	@Failure	500	{object}	string
//
//	@Router		/api/submit [put]
func PutSubmit(
	ctx context.Context,
	db *bun.DB,
	rdb *redis.Client,
	mc *minio.Client,
	problems utils.ProblemRepository,
	testcases map[string]*template.Template,
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
		var input submitInput
		if err := json.Unmarshal(b, &input); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "json unmarshal failed",
				OriginError: err,
			}
		}
		problem, ok := problems[input.ProblemId]
		if !ok {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "problem not exist",
			}
		}

		resource, ok := req.Context().Value(
			middlewares.UserResourceType("")).(models.UserResource)
		if !ok {
			return middlewares.HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "user resource missing",
			}
		}

		submission := models.Submission{
			ProblemName: problem.Name,
			ProblemId:   input.ProblemId,

			CsId:     resource.CsId,
			SubmitAt: time.Now().Unix(),
			Args:     make(map[string]string),

			Score: models.UserScore{
				Total:  0,
				Stages: make(map[string]int64),
			},
		}
		fmt.Fprintf(os.Stderr, "submission: %+v\n", submission)

		for key, val := range resource.Map() {
			submission.Args[key] = val
		}

		err = db.NewInsert().Model(&submission).Scan(ctx)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "submission insertion failed",
				OriginError: err,
			}
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
				err = utils.RunStage(ctx, rdb, mc, problems, testcases, submission, name)
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
