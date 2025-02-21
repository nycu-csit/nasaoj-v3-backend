package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/uptrace/bunrouter"
	"golang.org/x/oauth2"
)

// GetLogin godoc
//
//	@Description	OAuth login endpoint
//	@Header			302 {} string "AuthCodeURL(state)"
//	@Response		302 {} string "302 Found"
//	@Router			/auth/login [get]
func GetLogin(config oauth2.Config) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /api/login\n")
		var expire = time.Now().Add(365 * 24 * time.Hour)

		b := make([]byte, 16)
		rand.Read(b)
		state := base64.URLEncoding.EncodeToString(b)
		http.SetCookie(w, &http.Cookie{
			Name:    "oauthstate",
			Value:   state,
			Expires: expire,
		})

		w.Header().Set("Location", config.AuthCodeURL(state))
		w.WriteHeader(http.StatusFound)
		return nil
	}
}
