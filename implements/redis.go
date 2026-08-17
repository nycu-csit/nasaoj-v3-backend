package implements

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
)

const (
	workerStatusKey     = "worker-stat"
	workerJobsKeyPrefix = "worker-jobs-"
)

func resourceGroupKey(job *models.Job) string {
	return fmt.Sprintf("rg-%d-%s", job.SubId, job.ResourceGroup)
}

// Redis backs JobQueue (resource-group locks) and WorkerStatusStorage
// (worker status/lease bookkeeping) in multi-node mode, where that state
// must be shared across backend/worker processes. Token storage and
// problem/tool/score caching live in implements.Postgres; Redis is only
// consulted there as a cache layer in front of Postgres.
type Redis struct {
	rdb *redis.Client
}

func NewRedis(rdb *redis.Client) *Redis {
	return &Redis{
		rdb: rdb,
	}
}

func (self *Redis) GetWorkers(
	ctx context.Context,
) (models.WorkerStatusMap, error) {
	res, err := self.rdb.HGetAll(ctx, workerStatusKey).Result()
	if err != nil {
		return models.WorkerStatusMap{}, err
	}

	ret := make(models.WorkerStatusMap)
	for name, raw := range res {

		var status models.WorkerStatus
		if err := json.Unmarshal([]byte(raw), &status); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to unmarshal json",
					zap.Error(err))
			return models.WorkerStatusMap{}, err
		}

		ret[name] = status
	}

	return ret, nil
}

func (self *Redis) UpdateWorkerStatus(
	ctx context.Context,
	name string,
	ip string,
	tags []string,
) error {
	status := models.WorkerStatus{
		Ip:         ip,
		Tags:       tags,
		LastAccess: time.Now().Unix(),
	}
	_, err := self.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		b, err := json.Marshal(status)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to marshal to json",
					zap.Error(err))
			return err
		}

		pipe.HSet(ctx, workerStatusKey, name, string(b[:])).Result()
		return nil
	})
	return err
}

func (self *Redis) UpdateLastAccess(
	ctx context.Context,
	name string,
) error {
	if len(name) == 0 {
		tracing.Logger.Ctx(ctx).
			Error("worker name cannot be empty")
		return types.WrongFormatError
	}
	_, err := self.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		res, err := pipe.HGet(ctx, workerStatusKey, name).Result()
		if err != nil {
			return err
		}

		var status models.WorkerStatus
		if len(res) != 0 {
			if err := json.Unmarshal([]byte(res), &status); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to unmarshal json",
						zap.String("content", res),
						zap.Error(err))
				return err
			}
		}

		status.LastAccess = time.Now().Unix()

		b, err := json.Marshal(status)
		if err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to marshal to json",
					zap.Error(err))
			return err
		}

		_, err = pipe.HSet(ctx, workerStatusKey, name, string(b[:])).Result()
		return err
	})
	return err
}

func (self *Redis) DeleteWorker(
	ctx context.Context,
	name string,
) error {
	_, err := self.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		pipe.HDel(ctx, workerStatusKey, name)
		pipe.Del(ctx, workerJobsKeyPrefix+name)
		return nil
	})
	return err
}

func (self *Redis) AddWorkerJob(
	ctx context.Context,
	workerName string,
	jobId string,
) error {
	_, err := self.rdb.HSet(ctx, workerJobsKeyPrefix+workerName, jobId, "1").Result()
	return err
}

func (self *Redis) RemoveWorkerJob(
	ctx context.Context,
	workerName string,
	jobId string,
) error {
	_, err := self.rdb.HDel(ctx, workerJobsKeyPrefix+workerName, jobId).Result()
	return err
}

func (self *Redis) GetWorkerJobs(
	ctx context.Context,
	workerName string,
) ([]string, error) {
	res, err := self.rdb.HKeys(ctx, workerJobsKeyPrefix+workerName).Result()
	if err != nil {
		return nil, err
	}
	return res, nil
}

const cancelledSubsKey = "cancelled-subs"

func (self *Redis) AddCancelledSubmissions(
	ctx context.Context,
	subIds []int64,
) error {
	if len(subIds) == 0 {
		return nil
	}
	members := make([]interface{}, len(subIds))
	for i, id := range subIds {
		members[i] = id
	}
	_, err := self.rdb.SAdd(ctx, cancelledSubsKey, members...).Result()
	return err
}

func (self *Redis) GetCancelledSubmissions(
	ctx context.Context,
	subIds []int64,
) ([]int64, error) {
	if len(subIds) == 0 {
		return nil, nil
	}
	var cancelled []int64
	for _, subId := range subIds {
		exists, err := self.rdb.SIsMember(ctx, cancelledSubsKey, subId).Result()
		if err != nil {
			return nil, err
		}
		if exists {
			cancelled = append(cancelled, subId)
		}
	}
	return cancelled, nil
}

func (self *Redis) RemoveCancelledSubmissions(
	ctx context.Context,
	subIds []int64,
) error {
	if len(subIds) == 0 {
		return nil
	}
	members := make([]interface{}, len(subIds))
	for i, id := range subIds {
		members[i] = id
	}
	_, err := self.rdb.SRem(ctx, cancelledSubsKey, members...).Result()
	return err
}

func (self *Redis) Lock(
	ctx context.Context,
	job *models.Job,
) error {
	// No locking needed if ResourceGroup is empty
	if job.ResourceGroup == "" {
		return nil
	}

	key := resourceGroupKey(job)
	expiry := time.Duration(job.Timeout+60) * time.Second

	// Use SETNX with expiry for simple distributed lock
	ok, err := self.rdb.SetNX(ctx, key, job.Id(), expiry).Result()
	if err != nil {
		return err
	}
	if !ok {
		// Key already exists, lock is held by another job
		return types.LockAcquisitionError
	}

	return nil
}

func (self *Redis) Unlock(
	ctx context.Context,
	job *models.Job,
) error {
	// No unlocking needed if ResourceGroup is empty
	if job.ResourceGroup == "" {
		return nil
	}

	key := resourceGroupKey(job)

	// Delete the lock key
	_, err := self.rdb.Del(ctx, key).Result()
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Warn("failed to unlock resource group mutex",
				zap.String("resourceGroup", job.ResourceGroup),
				zap.Int64("subId", job.SubId),
				zap.Error(err))
		return err
	}

	return nil
}
