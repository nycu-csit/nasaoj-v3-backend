package implements

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

// ObjectStorage implementation for MinIO backend
type MinIO struct {
	*minio.Client
}

func NewMinIO(mc *minio.Client) *MinIO {
	return &MinIO{mc}
}

func (self *MinIO) PutFile(
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

func (self *MinIO) GetFileURL(
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

func (self *MinIO) GetFile(
	ctx context.Context,
	filename string,
) ([]byte, error) {
	buf, err := self.GetObject(ctx, viper.GetString("minio-bucket"),
		filename, minio.GetObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == minio.NoSuchKey {
			return []byte{}, types.FileNotExistError
		}
		return []byte{}, err
	}
	b, err := io.ReadAll(buf)
	if err != nil {
		return []byte{}, err
	}
	return b, nil
}

func (self *MinIO) ExistFile(
	ctx context.Context,
	filename string,
) (bool, error) {
	_, err := self.StatObject(ctx, viper.GetString("minio-bucket"),
		filename, minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code == minio.NoSuchKey {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
