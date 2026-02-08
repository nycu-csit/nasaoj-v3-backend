package implements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

const (
	tokenKey = "token"

	workerStatusKey     = "worker-stat"
	workerJobsKeyPrefix = "worker-jobs-"
)

func queueKey(
	tag string,
) string {
	return "queue-" + tag
}

func retryQueueKey(
	tag string,
) string {
	return "retry-queue-" + tag
}

func resourceGroupKey(job *models.Job) string {
	return fmt.Sprintf("rg-%d-%s", job.SubId, job.ResourceGroup)
}

type Redis struct {
	rdb *redis.Client
	rs  *redsync.Redsync
}

func NewRedis(rdb *redis.Client) *Redis {
	return &Redis{
		rdb: rdb,
		rs:  redsync.New(goredis.NewPool(rdb)),
	}
}

func (self *Redis) GetJob(
	ctx context.Context,
	tag string,
) (models.Job, error) {
	jobkey, err := self.rdb.RPop(ctx, queueKey(tag)).Result()
	if err != nil {
		return models.Job{}, err
	}

	jobRaw, err := self.rdb.Get(ctx, jobkey).Result()
	if err != nil {
		return models.Job{}, err
	}

	var job models.Job
	err = json.Unmarshal([]byte(jobRaw), &job)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to unmarshal json to job",
				zap.Error(err))
		return models.Job{}, err
	}

	return job, nil
}

func (self *Redis) PutJob(
	ctx context.Context,
	job models.Job,
	tag string,
) error {
	b, err := json.Marshal(job)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal job to json",
				zap.Error(err))
		return err
	}

	_, err = self.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		// Job schedule states
		_ = pipe.Set(ctx, job.Id(), b, 0)
		_ = pipe.LPush(ctx, queueKey(tag), job.Id())
		return nil
	})
	return err
}

func (self *Redis) SetJobCnt(
	ctx context.Context,
	stageId string,
	cnt int,
) error {
	_, err := self.rdb.HSet(ctx, "jobcnt", stageId, cnt).Result()
	return err
}

func (self *Redis) SetDepCnt(
	ctx context.Context,
	stageId string,
	cnt int,
) error {
	_, err := self.rdb.HSet(ctx, "jobdep", stageId, cnt).Result()
	return err
}

func (self *Redis) JobDone(
	ctx context.Context,
	stageId string,
) (int64, error) {
	return self.rdb.HIncrBy(ctx, "jobcnt", stageId, -1).Result()
}

func (self *Redis) StageDone(
	ctx context.Context,
	stageId string,
) (int64, error) {
	return self.rdb.HIncrBy(ctx, "jobdep", stageId, -1).Result()
}

func (self *Redis) RemoveJob(
	ctx context.Context,
	jobId string,
	tag string,
) error {
	_, err := self.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		_ = pipe.Del(ctx, jobId)
		_ = pipe.LRem(ctx, queueKey(tag), 0, jobId)
		return nil
	})
	return err
}

func (self *Redis) GetTokens(
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

func (self *Redis) GetPrivs(
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
	if !ret.IsEnabled {
		return models.Privileges{}, nil
	}
	return ret.Privs, nil
}

func (self *Redis) SetAdminToken(
	ctx context.Context,
	token string,
) error {
	h := sha256.Sum256([]byte(token))
	key := hex.EncodeToString(h[:])

	b, err := json.Marshal(models.Token{
		Privs:     models.NewAdminPrivs(0),
		IsEnabled: true,
		CreatedAt: time.Now().Unix(),
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

func (self *Redis) GenerateTokenWithoutExpire(
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
		CreatedAt: time.Now().Unix(),
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

func (self *Redis) GenerateToken(
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
		CreatedAt: time.Now().Unix(),
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

func (self *Redis) modifyToken(
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

func (self *Redis) DisableToken(
	ctx context.Context,
	key string,
) error {
	err := self.modifyToken(ctx, key, func(token models.Token) models.Token {
		token.IsEnabled = false
		return token
	})
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to disable token",
				zap.String("key", key),
				zap.Error(err))
		return err
	}
	return nil
}

func (self *Redis) EnableToken(
	ctx context.Context,
	key string,
) error {
	return self.modifyToken(ctx, key, func(token models.Token) models.Token {
		token.IsEnabled = true
		return token
	})
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

func (self *Redis) PutRetryJob(
	ctx context.Context,
	job models.Job,
	tag string,
) error {
	b, err := json.Marshal(job)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to marshal job to json",
				zap.Error(err))
		return err
	}

	_, err = self.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		_ = pipe.Set(ctx, job.Id(), b, 0)
		_ = pipe.LPush(ctx, retryQueueKey(tag), job.Id())
		return nil
	})
	return err
}

func (self *Redis) GetRetryJob(
	ctx context.Context,
	tag string,
) (models.Job, error) {
	jobkey, err := self.rdb.RPop(ctx, retryQueueKey(tag)).Result()
	if err != nil {
		return models.Job{}, err
	}

	jobRaw, err := self.rdb.Get(ctx, jobkey).Result()
	if err != nil {
		return models.Job{}, err
	}

	var job models.Job
	err = json.Unmarshal([]byte(jobRaw), &job)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to unmarshal json to job",
				zap.Error(err))
		return models.Job{}, err
	}

	return job, nil
}
