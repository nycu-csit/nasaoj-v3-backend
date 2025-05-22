package utils

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/template"
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
	testcases map[string]*template.Template,
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
		tw := tar.NewWriter(&tarbuf)
		for _, content := range contents {
			for _, fileName := range content.Files {
				testTmpl, ok := testcases[fileName]
				if !ok {
					return fmt.Errorf("testcase %s not exist\n", fileName)
				}

				b := bytes.NewBufferString("")
				if err := testTmpl.Execute(b, args); err != nil {
					return fmt.Errorf("template execute failed, %w\n", err)
				}

				hdr := &tar.Header{
					Name: fileName,
					Mode: 0600,
					Size: int64(b.Len()),
				}
				if err := tw.WriteHeader(hdr); err != nil {
					return fmt.Errorf("tar header write failed, %w\n", err)
				}
				if _, err := tw.Write(b.Bytes()); err != nil {
					return fmt.Errorf("tar body write failed, %w\n", err)
				}
			}
		}
		if err := tw.Close(); err != nil {
			return fmt.Errorf("tar writer close failed, %w\n", err)
		}
		_, err := mc.PutObject(ctx, env.MINIO_BUCKET,
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
