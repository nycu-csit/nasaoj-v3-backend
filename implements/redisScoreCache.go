package implements

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type RedisScoreCache struct {
	rdb *redis.Client
}

func NewRedisScoreCache(rdb *redis.Client) *RedisScoreCache {
	return &RedisScoreCache{rdb: rdb}
}

func (self *RedisScoreCache) Id(
	problem models.Problem,
	stageName string,
) string {
	return fmt.Sprintf("score-%d-%s", problem.ProblemId, stageName)
}

func (self *RedisScoreCache) Put(
	ctx context.Context,
	problem models.Problem,
	stageName string,
	csId string,
	score int64,
) error {
	_, err := self.rdb.HSet(ctx, self.Id(problem, stageName),
		csId, score).Result()
	return err
}
