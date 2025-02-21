package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
)

// GetScores
//
//	@Param		Cookie	header	string	false	"jwt"
//	@Success	200		{}		json
//	@Router		/api/scores [get]
func GetScores(ctx context.Context, rdb *redis.Client) func(http.ResponseWriter, bunrouter.Request) error {
	//var secretKey = []byte(os.Getenv("JWT_SECRET"))

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/scores\n")

		return nil
	}
}
