package handlers

import (
	"context"
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

// PutSubmit
//
// @Security ApiKeyAuth
// @Param		ProblemName	body	string	true	"Submit problem name"
// @Accept		text/plain
// @Success	200	{object} models.Submission
// @Router		/api/submit [put]
func PutSubmit(
	ctx context.Context,
	db *bun.DB,
	rdb *redis.Client,
	problems map[string]models.Problem,
	testcases map[string]*template.Template,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "PUT /api/submit\n")

		b, err := io.ReadAll(req.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read payload error\n")
			return err
		}
		problem, ok := problems[string(b[:])]
		if !ok {
			return fmt.Errorf("problem not exist\n")
		}

		resource, ok := req.Context().Value(
			middlewares.UserResourceType("")).(models.UserResource)
		if !ok {
			return fmt.Errorf("user resource missing\n")
		}

		submission := models.Submission{
			ProblemName: problem.Name,

			CsId:     resource.CsId,
			SubmitAt: time.Now().Unix(),
			Args:     make(map[string]string),

			Score: 0,
		}
		fmt.Fprintf(os.Stderr, "submission: %+v\n", submission)

		for key, val := range resource.Map() {
			submission.Args[key] = val
		}

		err = db.NewInsert().Model(&submission).Scan(ctx)
		if err != nil {
			return fmt.Errorf("submission insert failed%s\n", err)
		}
		subId := submission.Id

		stageDeps := make(map[string]int64)
		for name, stage := range problem.Stages {
			_, err = rdb.HSet(ctx, "jobcnt", fmt.Sprintf("%d-%s", subId, name),
				len(stage.Testcases)).Result()
			if err != nil {
				return fmt.Errorf("redis set failed, %s\n", err)
			}
			for _, next := range stage.Before {
				stageDeps[next] += 1
			}
		}
		fmt.Fprintf(os.Stderr, "stageDeps: %+v\n", stageDeps)

		for name, _ := range problem.Stages {
			if _, exist := stageDeps[name]; !exist {
				err = utils.RunStage(ctx, rdb, problems, testcases, submission, name)
				if err != nil {
					return err
				}
			} else {
				_, err = rdb.HSet(ctx, "depcnt", fmt.Sprintf("%d-%s", subId, name),
					stageDeps[name]).Result()
				if err != nil {
					return fmt.Errorf("redis set failed, %s\n", err)
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, submission)
		return nil
	}
}
