package implements

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type RedisWorkerStatusStorage struct {
	rdb *redis.Client
}

var (
	workerTagsKey       = "worker-tags"
	workerLastAccessKey = "worker-last-access"
)

func NewRedisWorkerStatusStorage(
	rdb *redis.Client,
) *RedisWorkerStatusStorage {
	return &RedisWorkerStatusStorage{rdb: rdb}
}

func (self *RedisWorkerStatusStorage) GetWorkers(
	ctx context.Context,
) (models.WorkerStatusMap, error) {
	tagsRes, err := self.rdb.HGetAll(ctx, workerTagsKey).Result()
	if err != nil {
		return models.WorkerStatusMap{},
			fmt.Errorf("redis HGETALL failed, %w", err)
	}
	lastAccessRes, err := self.rdb.HGetAll(ctx, workerLastAccessKey).Result()
	if err != nil {
		return models.WorkerStatusMap{},
			fmt.Errorf("redis HGETALL failed, %w", err)
	}

	ret := make(models.WorkerStatusMap)
	for ip, tagsStr := range tagsRes {
		lastAccessStr, ok := lastAccessRes[ip]
		if !ok {
			continue
		}

		var tags []string
		if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
			return models.WorkerStatusMap{},
				fmt.Errorf("json unmarshal failed, %w", err)
		}
		var lastAccess int64
		if err := json.Unmarshal([]byte(lastAccessStr), &lastAccess); err != nil {
			return models.WorkerStatusMap{},
				fmt.Errorf("json unmarshal failed, %w", err)
		}

		ret[ip] = models.WorkerStatus{
			Tags:       tags,
			LastAccess: lastAccess,
		}
	}

	return ret, nil
}

func (self *RedisWorkerStatusStorage) UpdateWorkerStatus(
	ctx context.Context,
	ip string,
	tags []string,
) error {
	_, err := self.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		b, err := json.Marshal(tags)
		if err != nil {
			return fmt.Errorf("json marshal failed, %w", err)
		}

		if _, err := pipe.HSet(ctx, workerTagsKey,
			ip, string(b[:])).Result(); err != nil {
			return fmt.Errorf("redis HSET failed, %w", err)
		}
		if _, err := pipe.HSet(ctx, workerLastAccessKey,
			ip, time.Now().Unix()).Result(); err != nil {
			return fmt.Errorf("redis HSET failed, %w", err)
		}
		return nil
	})
	return err
}

func (self *RedisWorkerStatusStorage) UpdateLastAccess(
	ctx context.Context,
	ip string,
) error {
	if _, err := self.rdb.HSet(ctx, workerLastAccessKey,
		ip, time.Now().Unix()).Result(); err != nil {
		return fmt.Errorf("redis HSET failed, %w", err)
	}
	return nil
}
