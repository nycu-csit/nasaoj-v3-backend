package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
)

// PostReloadScores
//
//	@Success	200	{string} string "Success"
//	@Failure	401
//	@Router		/api/admin/reloadscores [post]
func PostReloadScores(
	ctx context.Context,
	db *bun.DB,
	rdb *redis.Client,
	problems utils.ProblemRepository,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "POST /api/admin/reloadscores\n")

		scoreTables, err := utils.LoadScores(ctx, db, problems)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "load scores failed",
				OriginError: err,
			}
		}

		for id, table := range scoreTables {
			rawTable := make(map[string]string)
			for key, score := range table {
				b, err := json.Marshal(score)
				if err != nil {
					return middlewares.HTTPError{
						StatusCode:  http.StatusInternalServerError,
						Message:     "score unmarshal failed",
						OriginError: err,
					}
				}
				rawTable[key] = string(b[:])
			}

			hkey := fmt.Sprintf("scores-%d", id)
			_, err := rdb.Pipelined(ctx, func(pipe redis.Pipeliner) error {
				_ = pipe.Del(ctx, hkey)
				for key, value := range rawTable {
					_ = pipe.HSet(ctx, hkey, key, value)
				}
				return nil
			})
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "redis pipeline failed while update score table",
					OriginError: err,
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Success")
		return nil
	}
}
