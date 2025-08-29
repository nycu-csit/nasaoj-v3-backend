package implements

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

type RedisWorkerStatusStorage struct {
	rdb *redis.Client
}

const (
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
		return models.WorkerStatusMap{}, err
	}
	lastAccessRes, err := self.rdb.HGetAll(ctx, workerLastAccessKey).Result()
	if err != nil {
		return models.WorkerStatusMap{}, err
	}

	ret := make(models.WorkerStatusMap)
	for ip, tagsStr := range tagsRes {
		lastAccessStr, ok := lastAccessRes[ip]
		if !ok {
			continue
		}

		var tags []string
		if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to unmarshal json to tags",
					zap.Error(err))
			return models.WorkerStatusMap{}, err
		}
		var lastAccess int64
		if err := json.Unmarshal([]byte(lastAccessStr), &lastAccess); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to unmarshal json to last access",
					zap.Error(err))
			return models.WorkerStatusMap{}, err
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
			tracing.Logger.Ctx(ctx).
				Error("failed to marshal tags to json",
					zap.Error(err))
			return err
		}

		if _, err := pipe.HSet(ctx, workerTagsKey,
			ip, string(b[:])).Result(); err != nil {
			return err
		}
		if _, err := pipe.HSet(ctx, workerLastAccessKey,
			ip, time.Now().Unix()).Result(); err != nil {
			return err
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
		return err
	}
	return nil
}

func (self *RedisWorkerStatusStorage) DeleteWorker(
	ctx context.Context,
	ip string,
) error {
	if _, err := self.rdb.HDel(ctx,
		workerTagsKey, ip).Result(); err != nil {
		return err
	}
	if _, err := self.rdb.HDel(ctx,
		workerLastAccessKey, ip).Result(); err != nil {
		return err
	}
	return nil
}
