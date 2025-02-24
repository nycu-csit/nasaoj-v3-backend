package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/uptrace/bunrouter"
)

// GetOutput
//
//	@Param		subId		query		number	true	"Submission id"
//	@Param		stage		query		string	true	"Stage name"
//	@Param		testcase	query		string	true	"Testcase name"
//	@Success	200			{string}	string	"Testcase result output"
//	@Router		/api/output [get]
func GetOutput(ctx context.Context, mc *minio.Client) func(http.ResponseWriter, bunrouter.Request) error {
	MINIO_BUCKET := os.Getenv("MINIO_BUCKET")

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/output\n")

		subId, err := strconv.Atoi(req.URL.Query().Get("subId"))
		if err != nil {
			return fmt.Errorf("subId cannot convert to integer, %s\n", err)
		}
		stageName := req.URL.Query().Get("stage")
		testName := req.URL.Query().Get("testcase")

		fileName := fmt.Sprintf("%d-%s-%s-output", subId, stageName, testName)
		fmt.Fprintf(os.Stderr, "fileName: %s\n", fileName)

		output, err := mc.GetObject(ctx, MINIO_BUCKET, fileName,
			minio.GetObjectOptions{})
		if err != nil {
			return fmt.Errorf("minio get output failed, %s\n", err)
		}
		defer output.Close()

		b, err := io.ReadAll(output)

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "%s\n", string(b[:]))

		return nil
	}
}
