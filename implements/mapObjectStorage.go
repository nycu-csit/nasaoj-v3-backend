package implements

import (
	"context"
	"fmt"
	"time"
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
	_ context.Context,
	filename string,
) ([]byte, error) {
	ret, ok := self.data[filename]
	if !ok {
		return []byte{}, fmt.Errorf("file not exist")
	}
	return ret, nil
}
