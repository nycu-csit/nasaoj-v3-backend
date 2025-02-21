package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func PutResult(ctx context.Context, db *bun.DB, mc *minio.Client) func(http.ResponseWriter, bunrouter.Request) error {
	MINIO_BUCKET := os.Getenv("MINIO_BUCKET")
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		var err error
		err = Auth(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Authorization failed\n")
			return nil
		}

		body, err := io.ReadAll(req.Body)
		if err != nil {
			fmt.Printf("IO failed\n")
			return nil
		}

		fmt.Printf("PUT /api/result\nbody: %s\n", body)

		var job models.Job
		err = json.Unmarshal(body, &job)
		if err != nil {
			fmt.Fprintf(os.Stderr, "json unmarshal failed\n")
			return nil
		}

		_, err = db.NewInsert().Model(&job).On("CONFLICT (id) DO NOTHING").Exec(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "db insert failed\n")
			return nil
		}

		_, err = mc.PutObject(ctx, MINIO_BUCKET,
			fmt.Sprintf("%d-testcase", job.Id), strings.NewReader(job.TestContent), int64(len(job.TestContent)), minio.PutObjectOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "minio put testcase failed: %v\n", err)
			return nil
		}

		_, err = mc.PutObject(ctx, MINIO_BUCKET,
			fmt.Sprintf("%d-output", job.Id), strings.NewReader(job.Output), int64(len(job.Output)), minio.PutObjectOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "minio put output failed: %v\n", err)
			return nil
		}

		w.WriteHeader(http.StatusOK)
		return nil
	}
}
