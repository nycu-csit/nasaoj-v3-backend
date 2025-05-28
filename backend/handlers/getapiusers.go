package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
)

// GetUsers
//
//	@Success	200	{object}	[]models.UserResource
//	@Failure	401	{object}	string	"not admin or not login"
//	@Router		/api/admin/users [get]
func GetUsers(
	ctx context.Context,
	db *bun.DB,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/admin/users\n")

		var ret []models.UserResource
		err := db.NewSelect().Model(&ret).Scan(ctx)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "query user resource failed",
				OriginError: err,
			}
		}

		bunrouter.JSON(w, ret)
		return nil
	}
}
