package implements

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
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
	ret := []models.UserScore{}
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

func (self *RedisScoreCache) Reload(
	ctx context.Context,
	db interfaces.Database,
	problems models.ProblemRepository,
) error {
	scoreTables, err := utils.LoadScores(ctx, db, problems)
	if err != nil {
		return fmt.Errorf("load scores failed, %w", err)
	}
	fmt.Fprintf(os.Stderr, "scoreTables: %+v\n", scoreTables)

	_, err = self.rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
		for problemId, table := range scoreTables {
			for csId, score := range table {
				for stageName, value := range score.Stages {
					_, err = pipe.HSet(ctx,
						fmt.Sprintf("score-%d-%s", problemId, stageName),
						csId, value).Result()
					fmt.Fprintf(os.Stderr, "reload score-%d-%s\n", problemId, stageName)
					if err != nil {
						return fmt.Errorf("redis HSET failed, %w", err)
					}
				}
			}
		}
		return nil
	})
	return err
}
