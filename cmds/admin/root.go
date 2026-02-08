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
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/cancel"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/deletes"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/disable"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/enable"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/exports"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/gen"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/get"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/imports"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cmds/admin/submit"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

var (
	httpServer *http.Server
)

func getLogin(
	client *resty.Client,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

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
			SetQueryParam("source", "cli").
			Get(url)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "GET /auth/login failed",
				OriginError: err,
			}
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Error("resp.IsSuccess failed",
					zap.String("body", string(resp.Body()[:])))
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "resp.IsSuccess failed",
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
			http.SetCookie(w, cookie)
		}

		return utils.Redirect(w, location)
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

		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "GET /auth/callback failed",
				OriginError: err,
			}
		}

		if !resp.IsSuccess() {
			tracing.Logger.Ctx(ctx).
				Error("resp.IsSuccess failed",
					zap.String("body", string(resp.Body()[:])))
			return middlewares.HTTPError{
				StatusCode: http.StatusBadRequest,
				Message:    "resp.IsSuccess failed",
			}
		}

		for _, value := range resp.Header().Values("Set-Cookie") {
			cookie, err := http.ParseSetCookie(value)
			if err != nil {
				return middlewares.HTTPError{
					StatusCode:  http.StatusInternalServerError,
					Message:     "Set-Cookie header parse failed",
					OriginError: err,
				}
			}
			client.SetCookie(cookie)
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "<script>window.close();</script>")
		go func() {
			if err := httpServer.Shutdown(ctx); err != nil {
				tracing.Logger.Ctx(ctx).
					Panic("failed to shutdown server",
						zap.Error(err))
				panic(err)
			}
		}()

		return nil
	}
}

func oidcHandler(ctx context.Context, client *resty.Client) error {
	router := bunrouter.New()
	authGroup := router.NewGroup("/auth").
		Use(middlewares.ErrorHandler).
		Use(middlewares.CacheControlHandler)
	authGroup.GET("/login", getLogin(client))
	authGroup.GET("/callback", getCallback(ctx, client))

	httpServer = &http.Server{
		Addr:    ":" + viper.GetString("oauth2-port"),
		Handler: http.Handler(router),
	}

	oauthURL, err := url.JoinPath(
		"http://127.0.0.1:"+viper.GetString("oauth2-port"),
		"/auth/login")
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Panic("failed to construct url",
				zap.Error(err))
		panic(err)
	}

	go func() {
		tracing.Logger.Ctx(ctx).
			Info("open url",
				zap.String("url", oauthURL))
		cmd := exec.Command("xdg-open", oauthURL)
		cmd.Start()
	}()

	if err := httpServer.ListenAndServe(); err != http.ErrServerClosed {
		tracing.Logger.Ctx(ctx).
			Panic("failed to start http server",
				zap.Error(err))
		panic(err)
	}
	return nil
}

var RootCmd = &cobra.Command{
	Use:   "admin",
	Short: "Admin CLI utilities",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		utils.ReadConfig(ctx)
		tracing.InitTracer("nasaoj-cli")

		tracing.Logger.Ctx(ctx).
			Info("using endpoint",
				zap.String("url", viper.GetString("endpoint")))

		client := resty.New()
		client.SetRedirectPolicy(resty.NoRedirectPolicy())
		client.SetRetryCount(3)
		client.AddRetryCondition(func(r *resty.Response, err error) bool {
			return err != nil || !r.IsSuccess()
		})

		if viper.GetString("access-token") == "" {
			// Use OIDC
			err := oidcHandler(ctx, client)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Panic("failed to handle oidc",
						zap.Error(err))
				panic(err)
			}
		} else {
			tracing.Logger.Ctx(ctx).
				Info("use access token")
			client.SetAuthToken(viper.GetString("access-token"))
		}

		fmt.Fprintf(os.Stderr, "===== PreRun End =====\n")

		ctx = context.WithValue(ctx, types.RestyClient(""), client)
		cmd.SetContext(ctx)
	},
}

func init() {
	RootCmd.PersistentFlags().
		Bool("no-read-config", false,
			"Enable to not to read config")
	RootCmd.PersistentFlags().
		String("endpoint",
			"http://127.0.0.1:8080",
			"The endpoint of the backend server")
	RootCmd.PersistentFlags().
		String("oauth2-port", "8389",
			"The port that oauth2 server listens to")
	RootCmd.PersistentFlags().
		String("access-token", "",
			"The access token to use instead of oidc")

	RootCmd.AddCommand(imports.RootCmd)
	RootCmd.AddCommand(exports.RootCmd)
	RootCmd.AddCommand(enable.RootCmd)
	RootCmd.AddCommand(disable.RootCmd)
	RootCmd.AddCommand(get.RootCmd)
	RootCmd.AddCommand(deletes.RootCmd)
	RootCmd.AddCommand(gen.RootCmd)
	RootCmd.AddCommand(submit.RootCmd)
	RootCmd.AddCommand(cancel.RootCmd)
}
