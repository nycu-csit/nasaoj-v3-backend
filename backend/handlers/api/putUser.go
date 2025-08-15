package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

// PutUser
//
//	@Param		request	body	map[string]string	true	"User resource"
//	@Accept		json
//	@Success	200	{object}	string	"success"
//	@Failure	401	{object}	string
//	@Router		/api/user [put]
func PutUser(
	db interfaces.Database,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()
		fmt.Fprintf(os.Stderr, "PUT /api/user\n")

		privs, ok := ctx.Value(types.Privileges("")).(models.Privileges)
		if !ok {
			return middlewares.HTTPError{
				StatusCode: http.StatusUnauthorized,
				Message:    "privs not exist",
			}
		}

		b, err := io.ReadAll(req.Body)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "read payload failed",
				OriginError: err,
			}
		}

		var inputArgs map[string]string
		if err := json.Unmarshal(b, &inputArgs); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "json unmarshal failed",
				OriginError: err,
			}
		}

		if err := db.InsertUserResources(ctx,
			[]models.UserResource{models.UserResource{
				CsId: privs.OwnerCsId,
				Args: inputArgs,
			}}); err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusInternalServerError,
				Message:     "insert user resources failed",
				OriginError: err,
			}
		}

		fmt.Fprintf(w, "success")
		return nil
	}
}
