package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

type ScoreOutput struct {
	models.UserScore
	CsId string `json:"csId"`
}

// GetAdminScores
//
//	@Param		problemId	query		int				true	"Problem ID"
//	@Success	200			{object}	[]ScoreOutput	"Scores"
//	@Failure	400			{object}	string			"Invalid problem id"
//	@Router		/api/admin/scores [get]
func getAdminScoresPlaceholder() {}

// GetScores
//
//	@Param		problemId	query		int				true	"Problem ID"
//	@Success	200			{object}	[]ScoreOutput	"Scores"
//	@Failure	400			{object}	string			"Invalid problem id"
//	@Router		/api/scores [get]
func GetScores(
	ctx context.Context,
	rdb *redis.Client,
	problems utils.ProblemRepository,
	isAdmin bool,
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

		rawScores, err := rdb.HGetAll(ctx,
			fmt.Sprintf("scores-%d", problemId)).Result()
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "redis HGETALL failed",
				OriginError: err,
			}
		}
		scores := []ScoreOutput{}
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
			scores = append(scores, ScoreOutput{
				UserScore: score,
				CsId:      csId,
			})
		}

		// Trim stages result if not admin
		if !isAdmin {
			for csId, _ := range scores {
				scores[csId].Stages = make(map[string]int64)
			}
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, scores)
		return nil
	}
}
