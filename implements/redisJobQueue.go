package implements

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
)

type RedisJobQueue struct {
	rdb *redis.Client
}

func NewRedisJobQueue(rdb *redis.Client) *RedisJobQueue {
	return &RedisJobQueue{rdb: rdb}
}

func (self *RedisJobQueue) QueueKey(tag string) string {
	return "queue-" + tag
}

func (self *RedisJobQueue) GetJob(
	ctx context.Context,
	tag string,
) (models.Job, error) {
	jobkey, err := self.rdb.RPop(ctx, self.QueueKey(tag)).Result()
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

func (self *RedisJobQueue) PutJob(
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
		_ = pipe.LPush(ctx, self.QueueKey(tag), job.Id())
		return nil
	})
	return err
}

func (self *RedisJobQueue) SetJobCnt(
	ctx context.Context,
	stageId string,
	cnt int,
) error {
	_, err := self.rdb.HSet(ctx, "jobcnt", stageId, cnt).Result()
	return err
}

func (self *RedisJobQueue) SetDepCnt(
	ctx context.Context,
	stageId string,
	cnt int,
) error {
	_, err := self.rdb.HSet(ctx, "jobdep", stageId, cnt).Result()
	return err
}

func (self *RedisJobQueue) JobDone(
	ctx context.Context,
	stageId string,
) (int64, error) {
	return self.rdb.HIncrBy(ctx, "jobcnt", stageId, -1).Result()
}

func (self *RedisJobQueue) StageDone(
	ctx context.Context,
	stageId string,
) (int64, error) {
	return self.rdb.HIncrBy(ctx, "jobdep", stageId, -1).Result()
}
