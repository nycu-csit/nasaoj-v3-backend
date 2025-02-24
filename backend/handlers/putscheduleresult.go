package handlers

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
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

func PutScheduleResult(
	ctx context.Context,
	db *bun.DB,
	mc *minio.Client,
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

		w.WriteHeader(http.StatusOK)
		return nil
	}
}
