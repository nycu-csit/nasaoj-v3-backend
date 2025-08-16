package admin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/admin/deletes"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/admin/disable"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/admin/enable"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/admin/exports"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/admin/gen"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/admin/get"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmd/admin/imports"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
)

var (
	httpServer *http.Server
)

func getLogin(
	client *resty.Client,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/auth/login")
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "url construction failed",
				OriginError: err,
			}
		}

		resp, err := client.R().
			SetQueryString("source=cli").
			Get(url)
		if resp.StatusCode() != http.StatusFound && err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "GET /auth/login failed",
				OriginError: err,
			}
		}

		location := resp.Header().Get("Location")
		if location == "" {
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "missing redirect location",
			}
		}
		for _, cookie := range resp.Cookies() {
			fmt.Fprintf(os.Stderr, "[INFO] apply cookie '%s'\n", cookie.Name)
			http.SetCookie(w, cookie)
		}

		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusFound)
		return nil
	}
}

func getCallback(
	ctx context.Context,
	client *resty.Client,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/auth/callback")
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "url construction failed",
				OriginError: err,
			}
		}

		resp, err := client.R().
			SetCookies(req.Cookies()).
			SetQueryString(req.URL.RawQuery + "&source=cli").
			Get(url)

		if resp.StatusCode() != http.StatusFound {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "GET /auth/callback failed",
				OriginError: err,
			}
		}

		// fmt.Printf("%##v\n", resp.Request)
		// fmt.Printf("%##v\n", resp.RawResponse)

		for _, value := range resp.Header().Values("Set-Cookie") {
			cookie, err := http.ParseSetCookie(value)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "Set-Cookie header parse failed",
					OriginError: err,
				}
			}
			fmt.Fprintf(os.Stderr, "[INFO] apply cookie '%s'\n", cookie.Name)
			client.SetCookie(cookie)
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<script>window.close();</script>")
		go func() {
			if err := httpServer.Shutdown(ctx); err != nil {
				fmt.Printf("shutdown server failed, %v", err)
			}
		}()

		return nil
	}
}

func oidcHandler(ctx context.Context, client *resty.Client) error {
	router := bunrouter.New()
	authGroup := router.NewGroup("/auth").
		Use(middlewares.ErrorHandler)
	authGroup.GET("/login", getLogin(client))
	authGroup.GET("/callback", getCallback(ctx, client))

	httpServer = &http.Server{
		Addr:    ":" + viper.GetString("oauth2-port"),
		Handler: http.Handler(router),
	}

	url, err := url.JoinPath(
		fmt.Sprintf("http://localhost:%s", viper.GetString("oauth2-port")),
		"/auth/login")
	if err != nil {
		return fmt.Errorf("url construction failed")
	}

	go func() {
		fmt.Printf("URL: %s\n", url)
		cmd := exec.Command("xdg-open", url)
		cmd.Start()
	}()

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		fmt.Printf("http server failed, %v\n", err)
	}
	return nil
}

var RootCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin CLI utilities",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		client := resty.New()
		client.SetRedirectPolicy(resty.NoRedirectPolicy())

		if viper.GetString("access-token") == "" {
			// Use OIDC
			err := oidcHandler(ctx, client)
			if err != nil {
				return fmt.Errorf("oidc failed, %w", err)
			}
		} else {
			client.SetHeader("Authorization",
				"Bearer "+viper.GetString("access-token"))
		}

		fmt.Fprintf(os.Stderr, "===== PreRun End =====\n")

		ctx = context.WithValue(ctx, types.RestyClient(""), client)
		cmd.SetContext(ctx)
		return nil
	},
}

func init() {
	RootCmd.PersistentFlags().
		StringP("endpoint", "e",
			"http://localhost:8080",
			"The endpoint of the backend server")
	RootCmd.PersistentFlags().
		String("oauth2-port", "8389", "The port that oauth2 server listens to")
	RootCmd.PersistentFlags().
		StringP("access-token", "a",
			"", "The access token to use instead of oidc")

	RootCmd.AddCommand(imports.RootCmd)
	RootCmd.AddCommand(exports.RootCmd)
	RootCmd.AddCommand(enable.RootCmd)
	RootCmd.AddCommand(disable.RootCmd)
	RootCmd.AddCommand(get.RootCmd)
	RootCmd.AddCommand(deletes.RootCmd)
	RootCmd.AddCommand(gen.RootCmd)
}
