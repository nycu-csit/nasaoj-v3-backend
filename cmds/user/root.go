package user

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.uber.org/zap"
)

var (
	httpServer *http.Server
)

// CLISession stores the session tokens for the CLI
type CLISession struct {
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`
}

func GetSessionPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "judge", "session.json")
}

func LoadSession() (*CLISession, error) {
	path := GetSessionPath()
	if path == "" {
		return nil, fmt.Errorf("cannot determine home directory")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var session CLISession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	return &session, nil
}

func SaveSession(session *CLISession) error {
	path := GetSessionPath()
	if path == "" {
		return fmt.Errorf("cannot determine home directory")
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

func getLogin(
	client *resty.Client,
) func(http.ResponseWriter, bunrouter.Request) error {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		ctx := req.Context()

		url, err := url.JoinPath(
			viper.GetString("endpoint"), "/auth/login/cssso")
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "url construction failed",
				OriginError: err,
			}
		}

		resp, err := client.R().
			SetQueryParam("source", "cli").
			Post(url)
		if err != nil {
			return middlewares.HTTPError{
				StatusCode:  http.StatusBadRequest,
				Message:     "POST /auth/login failed",
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
	session *CLISession,
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

		// Extract tokens from cookies
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

			// Store tokens in session
			switch cookie.Name {
			case "refresh_token":
				session.RefreshToken = cookie.Value
			case "access_token":
				session.AccessToken = cookie.Value
			}
		}

		// Save session to file
		if err := SaveSession(session); err != nil {
			tracing.Logger.Ctx(ctx).
				Warn("failed to save session",
					zap.Error(err))
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

func oidcHandler(ctx context.Context, client *resty.Client, session *CLISession) error {
	router := bunrouter.New()
	authGroup := router.NewGroup("/auth").
		Use(middlewares.ErrorHandler).
		Use(middlewares.CacheControlHandler)
	authGroup.GET("/login", getLogin(client))
	authGroup.GET("/callback", getCallback(ctx, client, session))

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

// setupClientWithSession configures the resty client with session cookies
// and adds a response hook to capture refreshed tokens
func SetupClientWithSession(ctx context.Context, client *resty.Client, session *CLISession) {
	// Set cookies for authentication
	if session.AccessToken != "" {
		client.SetCookie(&http.Cookie{
			Name:  "access_token",
			Value: session.AccessToken,
		})
	}
	if session.RefreshToken != "" {
		client.SetCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: session.RefreshToken,
		})
	}

	// Add response hook to capture refreshed tokens
	client.OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
		var updated bool
		for _, value := range resp.Header().Values("Set-Cookie") {
			cookie, err := http.ParseSetCookie(value)
			if err != nil {
				continue
			}

			switch cookie.Name {
			case "refresh_token":
				if cookie.Value != session.RefreshToken {
					session.RefreshToken = cookie.Value
					c.SetCookie(cookie)
					updated = true
				}
			case "access_token":
				if cookie.Value != session.AccessToken {
					session.AccessToken = cookie.Value
					c.SetCookie(cookie)
					updated = true
				}
			}
		}

		// Save updated session
		if updated {
			tracing.Logger.Ctx(ctx).
				Info("tokens refreshed, saving session")
			if err := SaveSession(session); err != nil {
				tracing.Logger.Ctx(ctx).
					Warn("failed to save refreshed session",
						zap.Error(err))
			}
		}
		return nil
	})
}

var RootCmd = &cobra.Command{
	Use:   "user",
	Short: "User related interface",
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

		session := &CLISession{}

		// Try to load existing session
		if viper.GetString("access-token") != "" {
			// Use provided access token
			tracing.Logger.Ctx(ctx).
				Info("use access token from flag")
			client.SetAuthToken(viper.GetString("access-token"))
		} else if existingSession, err := LoadSession(); err == nil && existingSession.RefreshToken != "" {
			// Use stored session (access token can be obtained via refresh token)
			tracing.Logger.Ctx(ctx).
				Info("use session from file")
			session = existingSession
			SetupClientWithSession(ctx, client, session)
		} else {
			// Use OIDC
			err := oidcHandler(ctx, client, session)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Panic("failed to handle oidc",
						zap.Error(err))
				panic(err)
			}
			// Setup client with new session
			SetupClientWithSession(ctx, client, session)
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

}
