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

type RedisTokenStorage struct {
	rdb *redis.Client
}

func NewRedisTokenStorage(rdb *redis.Client) *RedisTokenStorage {
	return &RedisTokenStorage{rdb: rdb}
}

func (self *RedisTokenStorage) GetTokens(
	ctx context.Context,
) ([]models.Token, error) {
	res, err := self.rdb.HGetAll(ctx, "token").Result()
	if err != nil {
		return []models.Token{},
			fmt.Errorf("HGETALL failed, %w", err)
	}

	ret := []models.Token{}
	for sha, payload := range res {
		var privs models.Privileges
		if err := json.Unmarshal([]byte(payload), &privs); err != nil {
			return []models.Token{},
				fmt.Errorf("json unmarshal failed, %w", err)
		}
		ret = append(ret, models.Token{
			SHA256: sha,
			Privs:  privs,
		})
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

	res, err := self.rdb.HGet(ctx, "token", key).Result()
	if err != nil {
		return models.Privileges{},
			fmt.Errorf("redis HGET failed, %w", err)
	}
	// fmt.Fprintf(os.Stderr, "[INFO] res: %s\n", res)

	var ret models.Privileges
	if err := json.Unmarshal([]byte(res), &ret); err != nil {
		return models.Privileges{},
			fmt.Errorf("json unmarshal failed, %w", err)
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

	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	b, err := json.Marshal(privs)
	if err != nil {
		return "", fmt.Errorf("json marshal failed, %w", err)
	}

	_, err = self.rdb.HSet(ctx, "token", key, string(b[:])).Result()
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

	b, err := json.Marshal(privs)
	if err != nil {
		return "", fmt.Errorf("json marshal failed, %w", err)
	}

	_, err = self.rdb.HSet(ctx, "token", key, string(b[:])).Result()
	if err != nil {
		return "", fmt.Errorf("redis HSET failed, %w", err)
	}
	_, err = self.rdb.HExpire(ctx, "token", duration, key).Result()
	if err != nil {
		return "", fmt.Errorf("redis HExpire failed, %w", err)
	}
	return token, nil
}

func (self *RedisTokenStorage) DisableToken(
	ctx context.Context,
	key string,
) error {
	_, err := self.rdb.HDel(ctx, "token", key).Result()
	return err
}
