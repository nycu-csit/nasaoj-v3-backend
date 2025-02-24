package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
)

type GetScoresPayload struct {
	CsId  string  `json:"csid"`
	Score float64 `json:"score"`
}

// GetScores
//
//	@Success	200	{object}	GetScoresPayload
//	@Router		/api/scores [get]
func GetScores(ctx context.Context, rdb *redis.Client) func(http.ResponseWriter, bunrouter.Request) error {
	//var secretKey = []byte(os.Getenv("JWT_SECRET"))

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/scores\n")

		// Use fake output for now
		ret := []GetScoresPayload{
			GetScoresPayload{
				CsId:  "111550013",
				Score: 87.87,
			},
		}

		w.WriteHeader(http.StatusOK)
		bunrouter.JSON(w, ret)
		return nil
	}
}
