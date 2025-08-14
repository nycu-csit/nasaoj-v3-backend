package implements

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

type RedisTokenStorage struct {
	rdb *redis.Client
}

func NewRedisTokenStorage(rdb *redis.Client) *RedisTokenStorage {
	return &RedisTokenStorage{rdb: rdb}
}

func (self *RedisTokenStorage) GetPrivs(
	ctx context.Context,
	token string,
) (models.Privileges, error) {
	var ret models.Privileges

	res, err := self.rdb.HGet(ctx, "token", token).Result()
	if err != nil {
		return ret, fmt.Errorf("redis HGET failed, %w", err)
	}

	if err := json.Unmarshal([]byte(res), &ret); err != nil {
		return ret, fmt.Errorf("json unmarshal failed, %w", err)
	}
	return ret, nil
}

func (self *RedisTokenStorage) GenerateTokenWithoutExpire(
	ctx context.Context,
	privs models.Privileges,
) (string, error) {
	token, err := utils.RandomString(48)
	if err != nil {
		return "", fmt.Errorf("string generate failed, %w", err)
	}

	b, err := json.Marshal(privs)
	if err != nil {
		return "", fmt.Errorf("json marshal failed, %w", err)
	}

	_, err = self.rdb.HSet(ctx, "token", token, string(b[:])).Result()
	if err != nil {
		return "", fmt.Errorf("redis HSET failed, %w", err)
	}
	return token, nil
}

func (self *RedisTokenStorage) GenerateToken(
	ctx context.Context,
	privs models.Privileges,
	duration time.Duration,
) (string, error) {
	token, err := self.GenerateTokenWithoutExpire(ctx, privs)
	if err != nil {
		return "", err
	}

	_, err = self.rdb.HExpire(ctx, "token", duration, token).Result()
	if err != nil {
		return "", fmt.Errorf("redis HExpire failed, %w", err)
	}
	return token, nil
}

func (self *RedisTokenStorage) DisableToken(
	ctx context.Context,
	token string,
) error {
	_, err := self.rdb.HDel(ctx, "token", token).Result()
	return err
}
