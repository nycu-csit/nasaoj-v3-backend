package handlers

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
)

// PutLogout
//
// @Response 302 {} string "302 Found"
// @Router /auth/logout [put]
func PutLogout(
	ctx context.Context,
	db interfaces.Database,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "PUT /api/logout\n")

		http.SetCookie(w, &http.Cookie{
			Name: "jwt",
		})

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
		return nil
	}
}
