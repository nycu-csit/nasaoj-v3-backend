package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func PutScheduleResult(
	ctx context.Context,
	db *bun.DB,
	rdb *redis.Client,
	mc *minio.Client,
	problems utils.ProblemRepository,
	testcases map[string]*template.Template,
) func(http.ResponseWriter, bunrouter.Request) error {
	MINIO_BUCKET := os.Getenv("MINIO_BUCKET")
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		var err error

		err = utils.Auth(req)
		if err != nil {
			return fmt.Errorf("Authorization failed, %s\n", err)
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			return fmt.Errorf("IO failed, %s\n", err)
		}

		fmt.Printf("PUT /api/result\nbody: %s\n", body)

		var job models.Job
		err = json.Unmarshal(body, &job)
		if err != nil {
			return fmt.Errorf("json unmarshal failed, %s\n", err)
		}

		_, err = db.NewInsert().Model(&job).Exec(ctx)
		if err != nil {
			return fmt.Errorf("db insert failed, %s\n", err)
		}

		submission := models.Submission{
			Id: job.SubId,
		}

		err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			err = tx.NewSelect().Model(&submission).WherePK().Scan(ctx)
			if err != nil {
				return fmt.Errorf("submission select failed, %s\n", err)
			}

			submission.Jobs = append(submission.Jobs, &job)
			_, err = tx.NewUpdate().Model(&submission).WherePK().Exec(ctx)
			if err != nil {
				return fmt.Errorf("submission job append failed, %s\n", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("submission update tx failed, %s\n", err)
		}

		_, err = mc.PutObject(ctx, MINIO_BUCKET,
			fmt.Sprintf("%s-testcase", job.Id()),
			strings.NewReader(job.TestContent),
			int64(len(job.TestContent)),
			minio.PutObjectOptions{})
		if err != nil {
			return fmt.Errorf("minio put testcase failed, %s\n", err)
		}

		_, err = mc.PutObject(ctx, MINIO_BUCKET,
			fmt.Sprintf("%s-output", job.Id()),
			strings.NewReader(job.Output),
			int64(len(job.Output)),
			minio.PutObjectOptions{})
		if err != nil {
			return fmt.Errorf("minio put output failed, %s\n", err)
		}

		jobcnt, err := rdb.HIncrBy(ctx, "jobcnt", job.Stage(), -1).Result()
		if err != nil {
			return fmt.Errorf("redis jobcnt HIncrBy failed, %s\n", err)
		}

		if jobcnt <= 0 {
			problem := problems[int(submission.ProblemId)]
			stage := problem.Stages[job.StageName]

			err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				err = tx.NewSelect().Model(&submission).WherePK().Scan(ctx)
				if err != nil {
					return fmt.Errorf("submission select failed, %s\n", err)
				}

				submission.Score += stage.Score
				_, err = tx.NewUpdate().Model(&submission).WherePK().Exec(ctx)
				if err != nil {
					return fmt.Errorf("submission score upd failed, %s\n", err)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("submission upd tx failed, %s\n", err)
			}

			for _, next := range stage.Before {
				jobdep, err := rdb.HIncrBy(ctx, "jobdep",
					fmt.Sprintf("%d-%s", job.SubId, job.StageName), -1).Result()
				if err != nil {
					return fmt.Errorf("redis jobdep HIncrBy failed, %s\n", err)
				}

				if jobdep <= 0 {
					err = utils.RunStage(ctx, rdb, problems, testcases, submission, next)
					if err != nil {
						return err
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		return nil
	}
}
