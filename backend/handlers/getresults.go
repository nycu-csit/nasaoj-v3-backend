package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bunrouter"
)

// GetResults
//
//	@Param	Cookie		header	string	true	"jwt"
//	@Param	problem	query	string	true	"Problem name"
//	@Router	/api/results [get]
func GetResults(ctx context.Context, rdb *redis.Client) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/results\n")

		problemName := req.URL.Query().Get("problem")

		return nil
	}
}
