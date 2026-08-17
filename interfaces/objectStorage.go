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

	// PutFileURL presigns a direct upload URL so a caller (the worker) can
	// PUT a file straight to object storage without proxying the bytes
	// through the backend.
	PutFileURL(
		ctx context.Context,
		filename string,
		timeout time.Duration,
	) (string, error)

	GetFile(
		ctx context.Context,
		filename string,
	) ([]byte, error)

	ExistFile(
		ctx context.Context,
		filename string,
	) (bool, error)
}
