package implements

import (
	"context"
	"fmt"
	"os"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type MapTokenStorage struct {
	data map[string]models.Privileges
}

func NewMapTokenStorage() *MapTokenStorage {
	return &MapTokenStorage{
		data: make(map[string]models.Privileges),
	}
}

func (self *MapTokenStorage) GetPrivs(
	_ context.Context,
	token string,
) (models.Privileges, error) {
	ret, ok := self.data[token]
	if !ok {
		return models.Privileges{},
			fmt.Errorf("token '%s' not exist", token)
	}
	return ret, nil
}

func (self *MapTokenStorage) GenerateToken(
	ctx context.Context,
	privs models.Privileges,
	_ time.Duration,
) (string, error) {
	fmt.Fprintf(os.Stderr,
		"[WARN] tokens with expiration are not supported in this mode")
	return self.GenerateTokenWithoutExpire(ctx, privs)
}

func (self *MapTokenStorage) GenerateTokenWithoutExpire(
	_ context.Context,
	privs models.Privileges,
) (string, error) {
	token, err := utils.RandomString(48)
	if err != nil {
		return "", fmt.Errorf("string generate failed, %w", err)
	}

	self.data[token] = privs
	return token, nil
}

func (self *MapTokenStorage) DisableToken(
	_ context.Context,
	token string,
) error {
	delete(self.data, token)
	return nil
}
