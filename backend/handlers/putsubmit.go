package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
)

// PutSubmit
//
//	@Param		Cookie		header	string	true	"jwt"
//	@Param		ProblemName	body	string	true	"Submit problem name"
//	@Accept		text/plain
//	@Success	200	{}	string "ok"
//	@Router		/api/submit [put]
func PutSubmit(ctx context.Context, rdb *redis.Client) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "PUT /api/submit\n")

		problemName := req.URL.Query().Get("problem")

		return nil
	}
}
