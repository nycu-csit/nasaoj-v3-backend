package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"golang.org/x/oauth2"
)

var (
	webProvider     *oidc.Provider
	webOAuth2Config oauth2.Config
	cliProvider     *oidc.Provider
	cliOAuth2Config oauth2.Config
)

// GetLogin godoc
//
//	@Param			source	query	string false	"Login source"	attribute(optional)
//	@Description	OAuth login endpoint
//	@Header			302	{}	string	"AuthCodeURL(state)"
//	@Response		302	{}	string	"302 Found"
//	@Router			/auth/login [get]
func GetLogin(
	ctx context.Context,
) func(http.ResponseWriter, bunrouter.Request) error {
	var err error
	webProvider, err = oidc.NewProvider(
		ctx, viper.GetString("web-oauth-provider"))
	if err != nil {
		panic(err)
	}
	webOAuth2Config = oauth2.Config{
		ClientID:     viper.GetString("web-oauth-client-id"),
		ClientSecret: viper.GetString("web-oauth-client-secret"),
		RedirectURL:  viper.GetString("web-oauth-redirect-url"),
		Endpoint:     webProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "csid", "groups"},
	}
	cliProvider, err = oidc.NewProvider(
		ctx, viper.GetString("cli-oauth-provider"))
	if err != nil {
		panic(err)
	}
	cliOAuth2Config = oauth2.Config{
		ClientID:     viper.GetString("cli-oauth-client-id"),
		ClientSecret: viper.GetString("cli-oauth-client-secret"),
		RedirectURL:  viper.GetString("cli-oauth-redirect-url"),
		Endpoint:     cliProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "csid", "groups"},
	}

	return func(w http.ResponseWriter, req bunrouter.Request) error {
		fmt.Fprintf(os.Stderr, "GET /auth/login\n")
		var expire = time.Now().Add(365 * 24 * time.Hour)

		b := make([]byte, 16)
		rand.Read(b)
		state := base64.URLEncoding.EncodeToString(b)
		http.SetCookie(w, &http.Cookie{
			Name:    "oauthstate",
			Value:   state,
			Expires: expire,
		})

		if req.URL.Query().Get("source") == "cli" {
			w.Header().Set("Location", cliOAuth2Config.AuthCodeURL(state))
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "success")
		} else {
			w.Header().Set("Location", webOAuth2Config.AuthCodeURL(state))
			w.WriteHeader(http.StatusFound)
			fmt.Fprintf(w, "302 Found")
		}
		return nil
	}
}
