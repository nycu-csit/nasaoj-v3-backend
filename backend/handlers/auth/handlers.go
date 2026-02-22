package auth

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"golang.org/x/oauth2"
)

type Handlers struct {
	db interfaces.Database

	webProvider     *oidc.Provider
	webOAuth2Config oauth2.Config
	webVerifier     *oidc.IDTokenVerifier
	cliProvider     *oidc.Provider
	cliOAuth2Config oauth2.Config
	cliVerifier     *oidc.IDTokenVerifier
}

func NewHandlers(
	db *implements.Postgres,
) *Handlers {
	ctx := context.Background()

	webProvider, err := oidc.NewProvider(
		ctx, viper.GetString("web-oauth-provider"))
	if err != nil {
		panic(err)
	}
	webOAuth2Config := oauth2.Config{
		ClientID:     viper.GetString("web-oauth-client-id"),
		ClientSecret: viper.GetString("web-oauth-client-secret"),
		RedirectURL:  viper.GetString("web-oauth-redirect-url"),
		Endpoint:     webProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "csid", "groups"},
	}
	webVerifier := webProvider.Verifier(&oidc.Config{
		ClientID: viper.GetString("web-oauth-client-id")})

	cliProvider, err := oidc.NewProvider(
		ctx, viper.GetString("cli-oauth-provider"))
	if err != nil {
		panic(err)
	}
	cliOAuth2Config := oauth2.Config{
		ClientID:     viper.GetString("cli-oauth-client-id"),
		ClientSecret: viper.GetString("cli-oauth-client-secret"),
		RedirectURL:  viper.GetString("cli-oauth-redirect-url"),
		Endpoint:     cliProvider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "csid", "groups"},
	}
	cliVerifier := cliProvider.Verifier(&oidc.Config{
		ClientID: viper.GetString("cli-oauth-client-id")})

	return &Handlers{
		db: db,

		webProvider:     webProvider,
		webOAuth2Config: webOAuth2Config,
		webVerifier:     webVerifier,
		cliProvider:     cliProvider,
		cliOAuth2Config: cliOAuth2Config,
		cliVerifier:     cliVerifier,
	}
}
