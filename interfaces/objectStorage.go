package interfaces

import (
	"context"
	"time"
)

type ObjectStorage interface {
	PutFile(
		ctx context.Context,
		filename string,
		b []byte,
	) error

	GetFileURL(
		ctx context.Context,
		filename string,
		timeout time.Duration,
	) (string, error)

	/*
		GetFileByURL(
			ctx context.Context,
			URL string,
		) ([]byte, error)
	*/
}
