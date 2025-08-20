package implements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

var (
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
		return map[string]models.Token{},
			fmt.Errorf("HGETALL failed, %w", err)
	}

	ret := make(map[string]models.Token)
	for sha, payload := range res {
		var token models.Token
		if err := json.Unmarshal([]byte(payload), &token); err != nil {
			return map[string]models.Token{},
				fmt.Errorf("json unmarshal failed, %w", err)
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
	// fmt.Fprintf(os.Stderr, "[INFO] key: %s\n", key)

	res, err := self.rdb.HGet(ctx, tokenKey, key).Result()
	if err != nil {
		return models.Privileges{},
			fmt.Errorf("redis HGET failed, %w", err)
	}
	// fmt.Fprintf(os.Stderr, "[INFO] res: %s\n", res)

	var ret models.Token
	if err := json.Unmarshal([]byte(res), &ret); err != nil {
		return models.Privileges{},
			fmt.Errorf("json unmarshal failed, %w", err)
	}
	return ret.Privs, nil
}

func (self *RedisTokenStorage) GenerateTokenWithoutExpire(
	ctx context.Context,
	privs models.Privileges,
) (string, error) {
	token, err := utils.RandomString(48)
	if err != nil {
		return "", fmt.Errorf("string generate failed, %w", err)
	}

	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	b, err := json.Marshal(models.Token{
		Privs:     privs,
		IsEnabled: true,
	})
	if err != nil {
		return "", fmt.Errorf("json marshal failed, %w", err)
	}

	_, err = self.rdb.HSet(ctx, tokenKey, key, string(b[:])).Result()
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
	token, err := utils.RandomString(48)
	if err != nil {
		return "", fmt.Errorf("string generate failed, %w", err)
	}

	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	b, err := json.Marshal(models.Token{
		Privs:     privs,
		IsEnabled: true,
	})
	if err != nil {
		return "", fmt.Errorf("json marshal failed, %w", err)
	}

	_, err = self.rdb.HSet(ctx, tokenKey, key, string(b[:])).Result()
	if err != nil {
		return "", fmt.Errorf("redis HSET failed, %w", err)
	}
	_, err = self.rdb.HExpire(ctx, tokenKey, duration, key).Result()
	if err != nil {
		return "", fmt.Errorf("redis HExpire failed, %w", err)
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
		return fmt.Errorf("redis HGET failed, %w", err)
	}

	var token models.Token
	if err := json.Unmarshal([]byte(res), &token); err != nil {
		// fmt.Fprintf(os.Stderr, "[DEBUG] %+##v\n", res)
		return fmt.Errorf("json unmarshal failed, %w", err)
	}

	token = f(token)
	b, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("json marshal failed, %w", err)
	}
	if _, err := self.rdb.HSet(
		ctx, tokenKey, key, string(b[:])).Result(); err != nil {
		return fmt.Errorf("redis HSET failed, %w", err)
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
