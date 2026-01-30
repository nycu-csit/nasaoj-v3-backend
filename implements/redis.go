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

	workerTagsKey       = "worker-tags"
	workerLastAccessKey = "worker-last-access"
)

func queueKey(
	tag string,
) string {
	return "queue-" + tag
}

type Redis struct {
	rdb *redis.Client
}

func NewRedis(rdb *redis.Client) *Redis {
	return &Redis{rdb: rdb}
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
	return self.modifyToken(ctx, key, func(token models.Token) models.Token {
		token.IsEnabled = false
		return token
	})
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

func (self *Redis) UpdateWorkerStatus(
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

func (self *Redis) UpdateLastAccess(
	ctx context.Context,
	ip string,
) error {
	if _, err := self.rdb.HSet(ctx, workerLastAccessKey,
		ip, time.Now().Unix()).Result(); err != nil {
		return err
	}
	return nil
}

func (self *Redis) DeleteWorker(
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
