package implements

import (
	"context"
	"fmt"
	"strconv"

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

func (self *RedisScoreCache) Get(
	ctx context.Context,
	problem models.Problem,
	isAdmin bool,
) ([]models.UserScore, error) {
	var ret []models.UserScore
	table := make(map[string]map[string]int64)

	for stageName := range problem.Stages {
		rawScores, err := self.rdb.HGetAll(ctx,
			self.Id(problem, stageName)).Result()
		if err != nil {
			return ret, fmt.Errorf("score get failed, %w", err)
		}

		for csId, rawScore := range rawScores {
			if _, ok := table[csId]; !ok {
				table[csId] = make(map[string]int64)
			}

			score, err := strconv.ParseInt(rawScore, 10, 64)
			if err != nil {
				return ret, fmt.Errorf("score parse string failed, %w", err)
			}

			table[csId][stageName] = score
		}
	}

	for csId, stage := range table {
		var totalScore int64 = 0
		for _, v := range stage {
			totalScore += v
		}

		if isAdmin {
			ret = append(ret, models.UserScore{
				CsId: csId,
				Score: models.Score{
					Score:  totalScore,
					Stages: stage,
				},
			})
		} else {
			ret = append(ret, models.UserScore{
				CsId: csId,
				Score: models.Score{
					Score:  totalScore,
					Stages: nil,
				},
			})
		}
	}

	return ret, nil
}
