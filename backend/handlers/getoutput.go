package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/uptrace/bunrouter"
)

// GetOutput
//
//	@Param		TestName	query		string	true	"Testcase Name"
//	@Success	200			{string}	string	"Testcase result stdout"
//	@Router		/api/output [get]
func GetOutput(ctx context.Context, mc *minio.Client) func(http.ResponseWriter, bunrouter.Request) error {
	MINIO_BUCKET := os.Getenv("MINIO_BUCKET")

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/output\n")

		mc.GetObject(ctx, MINIO_BUCKET,
			fmt.Sprintf("%d-output", TestName))

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")

		return nil
	}
}
