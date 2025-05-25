package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/env"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// GetScores
//
//	@Param		problemId	query		int					true	"Problem ID"
//	@Success	200			{object}	utils.ScoreTable	"Scores"
//	@Failure	400			{object}	string				"Invalid problem id"
//	@Failure	402			{object}	string				"jwt claim missing"
//	@Router		/api/scores [get]
func GetScores(
	ctx context.Context,
	rdb *redis.Client,
	problems utils.ProblemRepository,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/scores\n")

		problemId, err := strconv.ParseInt(
			req.URL.Query().Get("problemId"), 10, 64)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "invalid problem id",
				OriginError: err,
			}
		}

		jwtClaim, ok := req.Context().
			Value(middlewares.JWTKeyType("")).(models.JWTClaim)
		if !ok {
			return middlewares.HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "jwt claim missing",
			}
		}

		rawScores, err := rdb.HGetAll(ctx,
			fmt.Sprintf("scores-%d", problemId)).Result()
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "redis HGETALL failed",
				OriginError: err,
			}
		}
		scores := make(utils.ScoreTable)
		for csId, rawScore := range rawScores {
			var score models.UserScore
			err = json.Unmarshal([]byte(rawScore), &score)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "score unmarshal failed",
					OriginError: err,
				}
			}
			scores[csId] = score
		}

		// Trim stages result if not admin
		if !slices.Contains(jwtClaim.Groups, env.ADMIN_ROLE) {
			for csId, score := range scores {
				scores[csId] = models.UserScore{
					Total:  score.Total,
					Stages: make(map[string]int64),
				}
			}
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, scores)
		return nil
	}
}
