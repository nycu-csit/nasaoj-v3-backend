package handlers

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"slices"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
)

// GetProblems
//
// @Security ApiKeyAuth
// @Produce	json
// @Success	200	{object}	[]models.Problem
// @Router		/api/problems [get]
func GetProblems(problems utils.ProblemRepository) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/problems\n")

		b, err := json.Marshal(slices.Collect(maps.Values(problems)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "json marshal failed: %s\n", err)
			return nil
		}
		fmt.Fprintf(os.Stderr, "json: %s\n", string(b[:]))

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "%s", string(b[:]))

		return nil
	}
}
