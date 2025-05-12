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
	problems utils.ProblemRepository,
	testcases map[string]*template.Template,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "PUT /api/submit\n")

		b, err := io.ReadAll(req.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(os.Stderr, "read payload error\n")
			return err
		}
		var input submitInput
		if err := json.Unmarshal(b, &input); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return fmt.Errorf("json unmarshal error: %w\n", err)
		}
		problem, ok := problems[input.ProblemId]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return fmt.Errorf("problem not exist\n")
		}

		resource, ok := req.Context().Value(
			middlewares.UserResourceType("")).(models.UserResource)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return fmt.Errorf("user resource missing\n")
		}

		submission := models.Submission{
			ProblemName: problem.Name,
			ProblemId:   input.ProblemId,
			CsId:        resource.CsId,
			SubmitAt:    time.Now().Unix(),
			Args:        make(map[string]string),

			Score: 0,
		}
		fmt.Fprintf(os.Stderr, "submission: %+v\n", submission)

		for key, val := range resource.Map() {
			submission.Args[key] = val
		}

		err = db.NewInsert().Model(&submission).Scan(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return fmt.Errorf("submission insert failed%s\n", err)
		}
		subId := submission.Id

		stageDeps := make(map[string]int64)
		for name, stage := range problem.Stages {
			_, err = rdb.HSet(ctx, "jobcnt", fmt.Sprintf("%d-%s", subId, name),
				len(stage.Testcases)).Result()
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return fmt.Errorf("redis set failed, %s\n", err)
			}
			for _, next := range stage.Before {
				stageDeps[next] += 1
			}
		}
		fmt.Fprintf(os.Stderr, "stageDeps: %+v\n", stageDeps)

		for name := range problem.Stages {
			if _, exist := stageDeps[name]; !exist {
				err = utils.RunStage(ctx, rdb, problems, testcases, submission, name)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return err
				}
			} else {
				_, err = rdb.HSet(ctx, "depcnt", fmt.Sprintf("%d-%s", subId, name),
					stageDeps[name]).Result()
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					return fmt.Errorf("redis set failed, %s\n", err)
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
