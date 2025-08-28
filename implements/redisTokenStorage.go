package implements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

const (
	tokenKey = "token"
)

type RedisTokenStorage struct {
	rdb *redis.Client
}

func NewRedisTokenStorage(rdb *redis.Client) *RedisTokenStorage {
	return &RedisTokenStorage{rdb: rdb}
}

func (self *RedisTokenStorage) GetTokens(
	ctx context.Context,
) (map[string]models.Token, error) {
	res, err := self.rdb.HGetAll(ctx, "token").Result()
	if err != nil {
		return map[string]models.Token{}, err
	}

	ret := make(map[string]models.Token)
	for sha, payload := range res {
		var token models.Token
		if err := json.Unmarshal([]byte(payload), &token); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to unmarshal json to token",
					zap.Error(err))
			return map[string]models.Token{}, err
		}
		ret[sha] = token
	}
	return ret, nil
}

func (self *RedisTokenStorage) GetPrivs(
	ctx context.Context,
	token string,
) (models.Privileges, error) {
	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	res, err := self.rdb.HGet(ctx, tokenKey, key).Result()
	if err != nil {
		return models.Privileges{}, err
	}

	var ret models.Token
	if err := json.Unmarshal([]byte(res), &ret); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to unmarshal json to token",
				zap.Error(err))
		return models.Privileges{}, err
	}
	return ret.Privs, nil
}

func (self *RedisTokenStorage) SetAdminToken(
	ctx context.Context,
	token string,
) error {
	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	b, err := json.Marshal(models.Token{
		Privs:     models.NewAdminPrivs(""),
		IsEnabled: true,
	})
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal token to json",
				zap.Error(err))
		return err
	}

	_, err = self.rdb.HSet(ctx, tokenKey, key, string(b[:])).Result()
	if err != nil {
		return err
	}
	return nil
}

func (self *RedisTokenStorage) GenerateTokenWithoutExpire(
	ctx context.Context,
	privs models.Privileges,
) (string, error) {
	token, err := utils.RandomString(48)
	if err != nil {
		return "", err
	}

	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	b, err := json.Marshal(models.Token{
		Privs:     privs,
		IsEnabled: true,
	})
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal token to json",
				zap.Error(err))
		return "", err
	}

	_, err = self.rdb.HSet(ctx, tokenKey, key, string(b[:])).Result()
	if err != nil {
		return "", err
	}
	return token, nil
}

func (self *RedisTokenStorage) GenerateToken(
	ctx context.Context,
	privs models.Privileges,
	duration time.Duration,
) (string, error) {
	token, err := utils.RandomString(48)
	if err != nil {
		return "", err
	}

	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	b, err := json.Marshal(models.Token{
		Privs:     privs,
		IsEnabled: true,
	})
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal token to json",
				zap.Error(err))
		return "", err
	}

	_, err = self.rdb.HSet(ctx, tokenKey, key, string(b[:])).Result()
	if err != nil {
		return "", err
	}
	_, err = self.rdb.HExpire(ctx, tokenKey, duration, key).Result()
	if err != nil {
		return "", err
	}
	return token, nil
}

func (self *RedisTokenStorage) modifyToken(
	ctx context.Context,
	key string,
	f func(models.Token) models.Token,
) error {
	res, err := self.rdb.HGet(ctx, tokenKey, key).Result()
	if err != nil {
		return err
	}

	var token models.Token
	if err := json.Unmarshal([]byte(res), &token); err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to unmarshal json to token",
				zap.Error(err))
		return err
	}

	token = f(token)
	b, err := json.Marshal(token)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal token to json",
				zap.Error(err))
		return err
	}
	if _, err := self.rdb.HSet(
		ctx, tokenKey, key, string(b[:])).Result(); err != nil {
		return err
	}

	return nil
}

func (self *RedisTokenStorage) DisableToken(
	ctx context.Context,
	key string,
) error {
	return self.modifyToken(ctx, key, func(token models.Token) models.Token {
		token.IsEnabled = false
		return token
	})
}

func (self *RedisTokenStorage) EnableToken(
	ctx context.Context,
	key string,
) error {
	return self.modifyToken(ctx, key, func(token models.Token) models.Token {
		token.IsEnabled = true
		return token
	})
}
