package utils

import (
	"context"
	"os"

	"github.com/minio/minio-go/v7"
)

func InitMinIO(ctx context.Context, mc *minio.Client) error {
	MINIO_BUCKET := os.Getenv("MINIO_BUCKET")

	err := mc.MakeBucket(ctx, MINIO_BUCKET, minio.MakeBucketOptions{
		Region: "us-east-1",
	})
	if err != nil {
		exists, errBucketExists := mc.BucketExists(ctx, MINIO_BUCKET)
		if errBucketExists != nil || !exists {
			return errBucketExists
		}
	}

	return nil
}
