package implements

import (
	"bytes"
	"context"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

// ObjectStorage implementation for MinIO backend
type MinIOObjectStorage struct {
	*minio.Client
}

func NewMinIOObjectStorage(mc *minio.Client) *MinIOObjectStorage {
	return &MinIOObjectStorage{mc}
}

func (self MinIOObjectStorage) PutFile(
	ctx context.Context,
	filename string,
	b []byte,
) error {
	_, err := self.PutObject(ctx, viper.GetString("minio-bucket"),
		filename, bytes.NewReader(b), int64(len(b)), minio.PutObjectOptions{})
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to put file to minio",
				zap.Error(err))
		return err
	}

	return nil
}

func (self MinIOObjectStorage) GetFileURL(
	ctx context.Context,
	filename string,
	timeout time.Duration,
) (string, error) {
	fileURL, err := self.PresignedGetObject(ctx, viper.GetString("minio-bucket"),
		filename, timeout, nil)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to get presigned url from minio",
				zap.Error(err))
		return "", err
	}

	return fileURL.String(), nil
}
