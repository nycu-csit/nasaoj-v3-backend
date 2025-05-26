package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/env"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func RunStage(
	ctx context.Context,
	rdb *redis.Client,
	mc *minio.Client,
	problems ProblemRepository,
	testcases TestcaseRepository,
	submission models.Submission,
	stageName string,
) error {
	fmt.Fprintf(os.Stderr, "RunStage %s-%s\n", submission.ProblemName, stageName)

	problem := problems[submission.ProblemId]
	stage := problem.Stages[stageName]

	fmt.Fprintf(os.Stderr, "Stage: %+v\n", problem.Stages[stageName])

	for _, testcase := range stage.Testcases {
		fmt.Fprintf(os.Stderr, "testcase: %v\n", testcase)

		contents := []models.Content{
			problem.PreScript,
			stage.PreScript,
			testcase.Content,
			stage.PostScript,
			problem.PostScript,
		}

		jobScript := ""
		for _, content := range contents {
			for _, script := range content.Scripts {
				jobScript += script + "\n"
			}
		}

		job := models.Job{
			SubId:     submission.Id,
			StageName: stageName,
			TestName:  testcase.Name,

			Script: jobScript,

			HardTL: testcase.HardTL,
			SoftTL: testcase.SoftTL,
		}

		args := ArgsOverlay(
			submission.Args,
			problem.Args,
			stage.Args,
			testcase.Args,
		)

		var tarbuf bytes.Buffer
		err := WriteTar(&tarbuf, contents, testcases, args)
		if err != nil {
			return fmt.Errorf("write to tar failed, %w\n", err)
		}

		_, err = mc.PutObject(ctx, env.MINIO_BUCKET,
			fmt.Sprintf("%s.tar", job.Id()),
			bytes.NewReader(tarbuf.Bytes()),
			int64(tarbuf.Len()),
			minio.PutObjectOptions{},
		)
		if err != nil {
			return fmt.Errorf("minio put object failed, %w\n", err)
		}

		timeout := time.Duration(job.HardTL) * time.Second
		fileURL, err := mc.PresignedGetObject(ctx, env.MINIO_BUCKET,
			fmt.Sprintf("%s.tar", job.Id()), timeout, nil)
		if err != nil {
			return fmt.Errorf("minio generate presigned url failed, %w\n", err)
		}
		job.FileURL = fileURL.String()

		fmt.Fprintf(os.Stderr, "job: %+v\n", job)

		payload, err := json.Marshal(job)
		if err != nil {
			return fmt.Errorf("job JSON Marshal error, %w\n", err)
		}
		fmt.Fprintf(os.Stderr, "job %s payload: %s\n", job.Id(), string(payload[:]))
		_, err = rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
			// Job schedule states
			_ = pipe.Set(ctx, job.Id(), payload, 0)
			_ = pipe.LPush(ctx, fmt.Sprintf("queue-%s", problem.Tag), job.Id())

			// Job expiration states
			subJob := fmt.Sprintf("sub-job-%d", submission.Id)
			_ = pipe.HSet(ctx, subJob, job.Id(), "deadbeef")
			_ = pipe.HExpire(ctx, subJob, timeout, job.Id())
			return nil
		})
		if err != nil {
			return fmt.Errorf("redis pipeline operation failed, %w\n", err)
		}
	}

	return nil
}
