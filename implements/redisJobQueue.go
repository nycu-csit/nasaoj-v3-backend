package implements

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type RedisJobQueue struct {
	rdb *redis.Client
}

func NewRedisJobQueue(rdb *redis.Client) *RedisJobQueue {
	return &RedisJobQueue{rdb: rdb}
}

func (self *RedisJobQueue) QueueKey(tag models.TagType) string {
	return fmt.Sprintf("queue-%s", tag)
}

func (self *RedisJobQueue) GetJob(
	ctx context.Context,
	tag models.TagType,
) (models.Job, error) {
	jobkey, err := self.rdb.RPop(ctx, self.QueueKey(tag)).Result()
	if err != nil {
		return models.Job{}, fmt.Errorf("rdb RPOP failed or no job left in queue")
	}

	jobRaw, err := self.rdb.Get(ctx, jobkey).Result()
	if err != nil {
		return models.Job{}, fmt.Errorf("rdb GET failed")
	}

	var job models.Job
	err = json.Unmarshal([]byte(jobRaw), &job)
	if err != nil {
		return models.Job{}, fmt.Errorf("job json unmarshal failed")
	}

	return job, nil
}

func (self *RedisJobQueue) PutJob(
	ctx context.Context,
	job models.Job,
	tag models.TagType,
) error {
	b, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal job to json failed, %w", err)
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
