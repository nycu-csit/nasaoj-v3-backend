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
	"github.com/spf13/viper"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
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
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		var err error

		err = utils.Auth(req)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusUnauthorized,
				Message:     "not authorized",
				OriginError: err,
			}
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "data retrieve failed",
				OriginError: err,
			}
		}

		fmt.Fprintf(os.Stderr, "PUT /schedule/result\nbody: %s\n", body)

		var job models.Job
		err = json.Unmarshal(body, &job)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "json unmarshal failed",
				OriginError: err,
			}
		}

		subJob := fmt.Sprintf("sub-job-%d", job.SubId)
		subJobCnt, err := rdb.HDel(ctx, subJob, job.Id()).Result()
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "redis HDEL failed",
				OriginError: err,
			}
		}
		if subJobCnt <= 0 {
			return middlewares.HTTPError{
				StatusCode: http.StatusForbidden,
				Message: fmt.Sprintf("job expired or not exist, %v, %v",
					subJob, subJobCnt),
			}
		}

		submission := models.Submission{
			Id: job.SubId,
		}

		err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
			err = tx.NewSelect().Model(&submission).WherePK().Scan(ctx)
			if err != nil {
				return fmt.Errorf("submission select failed, %w\n", err)
			}

			if job.Status != models.StatusTimeout {
				job.Status = models.StatusFailed
			}
			_, err = tx.NewUpdate().Model(&models.Job{}).
				Set("status = ?", models.StatusSkip).
				Where("subid = ?", job.SubId).
				Where("status IN (?)", bun.In([]models.JobStatus{
					models.StatusPending,
				})).Exec(ctx)
			if err != nil {
				return fmt.Errorf("job status skip update failed, %w\n", err)
			}

			_, err = tx.NewUpdate().Model(&job).WherePK().Exec(ctx)
			if err != nil {
				return fmt.Errorf("job status failed update failed, %w\n", err)
			}
			// submission.Jobs = append(submission.Jobs, &job)
			// _, err = tx.NewUpdate().Model(&submission).WherePK().Exec(ctx)
			// if err != nil {
			// 	return fmt.Errorf("submission job append failed, %s\n", err)
			// }
			return nil
		})
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "submission update transaction failed",
				OriginError: err,
			}
		}

		_, err = mc.PutObject(ctx, viper.GetString("minio-bucket"),
			job.StdoutKey(), strings.NewReader(job.Stdout),
			int64(len(job.Stdout)), minio.PutObjectOptions{})
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "minio put stdout failed",
				OriginError: err,
			}
		}

		_, err = mc.PutObject(ctx, viper.GetString("minio-bucket"),
			job.StderrKey(), strings.NewReader(job.Stderr),
			int64(len(job.Stderr)), minio.PutObjectOptions{})
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "minio put stderr failed",
				OriginError: err,
			}
		}

		if job.Status != models.StatusSuccess {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "success")
			return nil
		}
		jobcnt, err := rdb.HIncrBy(ctx, "jobcnt", job.Stage(), -1).Result()
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "redis jobcnt HINCRBY failed",
				OriginError: err,
			}
		}

		if jobcnt <= 0 {
			problem := problems[submission.ProblemId]
			stage := problem.Stages[job.StageName]

			err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				err = tx.NewSelect().Model(&submission).WherePK().Scan(ctx)
				if err != nil {
					return fmt.Errorf("submission select failed, %s\n", err)
				}

				submission.Score.Score += stage.Score
				submission.Score.Stages[job.StageName] += stage.Score
				_, err = tx.NewUpdate().Model(&submission).WherePK().Exec(ctx)
				if err != nil {
					return fmt.Errorf("submission score upd to pg failed, %s\n", err)
				}

				b, err := json.Marshal(submission.Score)
				if err != nil {
					return fmt.Errorf("score marshal failed")
				}
				_, err = rdb.HSet(ctx, fmt.Sprintf("scores-%d", problem.ProblemId),
					submission.CsId, string(b[:])).Result()
				if err != nil {
					return fmt.Errorf("redis HSET failed while upd score")
				}
				return nil
			})
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "submission update transaction failed",
					OriginError: err,
				}
			}

			for _, next := range stage.Before {
				jobdep, err := rdb.HIncrBy(ctx, "jobdep",
					fmt.Sprintf("%d-%s", job.SubId, job.StageName), -1).Result()
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "redis jobdep HINCRBY failed",
						OriginError: err,
					}
				}

				if jobdep <= 0 {
					err = utils.RunStage(
						ctx, db, rdb, mc, problems, testcases, submission, next)
					if err != nil {
						return middlewares.HTTPError{
							StatusCode:  http.StatusInternalServerError,
							Message:     "RunStage failed",
							OriginError: err,
						}
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		return nil
	}
}
