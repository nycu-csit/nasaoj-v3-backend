package implements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type MapTokenStorage struct {
	data map[string]models.Token
}

func NewMapTokenStorage() *MapTokenStorage {
	return &MapTokenStorage{
		data: make(map[string]models.Token),
	}
}

func (self *MapTokenStorage) GetTokens(
	_ context.Context,
) (map[string]models.Token, error) {
	return self.data, nil
}

func (self *MapTokenStorage) GetPrivs(
	_ context.Context,
	token string,
) (models.Privileges, error) {
	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	ret, ok := self.data[key]
	if !ok {
		return models.Privileges{},
			fmt.Errorf("token '%s' not exist", token)
	}
	if !ret.IsEnabled {
		return models.Privileges{},
			fmt.Errorf("token is not enabled")
	}
	return ret.Privs, nil
}

func (self *MapTokenStorage) GenerateToken(
	ctx context.Context,
	privs models.Privileges,
	_ time.Duration,
) (string, error) {
	fmt.Fprintf(os.Stderr,
		"[WARN] tokens with expiration are not supported in this mode\n")
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

	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	self.data[key] = models.Token{
		Privs:     privs,
		IsEnabled: true,
	}
	return token, nil
}

func (self *MapTokenStorage) DisableToken(
	_ context.Context,
	key string,
) error {
	token, ok := self.data[key]
	if !ok {
		return fmt.Errorf("token not exist")
	}

	token.IsEnabled = false
	self.data[key] = token
	return nil
}

func (self *MapTokenStorage) EnableToken(
	_ context.Context,
	key string,
) error {
	token, ok := self.data[key]
	if !ok {
		return fmt.Errorf("token not exist")
	}

	token.IsEnabled = true
	self.data[key] = token
	return nil
}
