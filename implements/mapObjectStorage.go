package implements

import (
	"context"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

type MapObjectStorage struct {
	data    map[string][]byte
	timeout map[string]time.Duration
}

func NewMapObjectStorage() *MapObjectStorage {
	return &MapObjectStorage{
		data:    make(map[string][]byte),
		timeout: make(map[string]time.Duration),
	}
}

func (self *MapObjectStorage) PutFile(
	_ context.Context,
	filename string,
	b []byte,
) error {
	self.data[filename] = b
	return nil
}

func (self *MapObjectStorage) GetFileURL(
	_ context.Context,
	filename string,
	timeout time.Duration,
) (string, error) {
	self.timeout[filename] = timeout
	return filename, nil
}

func (self *MapObjectStorage) GetFileByURL(
	ctx context.Context,
	filename string,
) ([]byte, error) {
	ret, ok := self.data[filename]
	if !ok {
		tracing.Logger.Ctx(ctx).
			Error("file not exist",
				zap.String("file.name", filename))
		return []byte{}, types.FileNotExistError
	}
	return ret, nil
}
