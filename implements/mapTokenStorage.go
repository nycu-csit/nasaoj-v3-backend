package implements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
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
	ctx context.Context,
	token string,
) (models.Privileges, error) {
	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	ret, ok := self.data[key]
	if !ok {
		tracing.Logger.Ctx(ctx).
			Warn("token not exist",
				zap.String("token.sha256", key))
		return models.Privileges{}, types.TokenNotExistError
	}
	if !ret.IsEnabled {
		tracing.Logger.Ctx(ctx).
			Warn("token not enabled",
				zap.String("token.sha256", key))
		return models.Privileges{}, types.TokenNotEnabledError
	}
	return ret.Privs, nil
}

func (self *MapTokenStorage) SetAdminToken(
	_ context.Context,
	token string,
) error {
	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	self.data[key] = models.Token{
		Privs:     models.NewAdminPrivs(""),
		IsEnabled: true,
	}
	return nil
}

func (self *MapTokenStorage) GenerateToken(
	ctx context.Context,
	privs models.Privileges,
	_ time.Duration,
) (string, error) {
	tracing.Logger.Ctx(ctx).
		Warn("tokens with expiration are not supported in this mode")
	return self.GenerateTokenWithoutExpire(ctx, privs)
}

func (self *MapTokenStorage) GenerateTokenWithoutExpire(
	_ context.Context,
	privs models.Privileges,
) (string, error) {
	token, err := utils.RandomString(48)
	if err != nil {
		return "", err
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
	ctx context.Context,
	key string,
) error {
	token, ok := self.data[key]
	if !ok {
		tracing.Logger.Ctx(ctx).
			Error("token not exist",
				zap.String("token.sha256", key))
		return types.TokenNotExistError
	}

	token.IsEnabled = false
	self.data[key] = token
	return nil
}

func (self *MapTokenStorage) EnableToken(
	ctx context.Context,
	key string,
) error {
	token, ok := self.data[key]
	if !ok {
		tracing.Logger.Ctx(ctx).
			Error("token not exist",
				zap.String("token.sha256", key))
		return types.TokenNotExistError
	}

	token.IsEnabled = true
	self.data[key] = token
	return nil
}
