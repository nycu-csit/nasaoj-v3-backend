package cmd

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
	"golang.org/x/oauth2"

	_ "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/docs"
)

//	@title			NASAOJ v3
//	@version		2.0
//	@termsOfService	http://swagger.io/terms/

//	@license.name	WTFPL
//	@BasePath		/

// @Securitydefinitions.apikey	ApiKeyAuth
// @In							header
// @Name						Cookie
// @Description				jwt=value, get it at /auth/login
var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "NASAOJ backend server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		provider, err := oidc.NewProvider(ctx, "https://oauth.cs.nycu.edu.tw")
		if err != nil {
			panic(err)
		}

		// Configure an OpenID Connect aware OAuth2 client.
		oauth2Config := oauth2.Config{
			ClientID:     viper.GetString("oauth-client-id"),
			ClientSecret: viper.GetString("oauth-client-secret"),
			RedirectURL:  viper.GetString("oauth-redirect-url"),

			// Discovery returns the OAuth2 endpoints.
			Endpoint: provider.Endpoint(),

			// "openid" is a required scope for OpenID Connect flows.
			Scopes: []string{oidc.ScopeOpenID, "csid", "groups"},
		}

		// Initialize DB instance
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
		db := bun.NewDB(sqldb, pgdialect.New())

		// Initialize Redis instance
		rdb := redis.NewClient(&redis.Options{
			Addr:     viper.GetString("redis-host"),
			Username: viper.GetString("redis-user"),
			Password: viper.GetString("redis-pass"),
			DB:       viper.GetInt("redis-db"),
		})

		// Initialize MinIO instance
		mc, err := minio.New(viper.GetString("minio-host"), &minio.Options{
			Creds: credentials.NewStaticV4(
				viper.GetString("minio-accesskey"),
				viper.GetString("minio-secretkey"),
				"",
			),
			Secure: viper.GetBool("minio-usessl"),
		})
		if err != nil {
			panic(err)
		}

		if err := utils.InitMinIO(ctx, mc); err != nil {
			panic(err)
		}

		// Load problems and testcases
		problems, err := utils.LoadProblems(viper.GetString("problems-dir"))
		if err != nil {
			panic(err)
		}
		testcases, err := utils.LoadTestcases(viper.GetString("testcases-dir"))
		if err != nil {
			panic(err)
		}

		// Initialize backend router
		backend := bunrouter.New()

		backend.GET("/swagger/*any",
			bunrouter.HTTPHandlerFunc(
				httpSwagger.Handler()))

		authGroup := backend.NewGroup("/auth").
			Use(middlewares.ErrorHandler)
		authGroup.GET("/login", handlers.GetLogin(oauth2Config))
		authGroup.GET("/callback", handlers.GetCallback(ctx, db,
			oauth2Config, provider.Verifier(&oidc.Config{
				ClientID: viper.GetString("oauth-client-id"),
			})))

		apiGroup := backend.NewGroup("/api").
			Use(middlewares.ErrorHandler).
			Use(middlewares.CORSHandler).
			Use(middlewares.CheckJWT(ctx, db))
		apiGroup.GET("/problems", handlers.GetProblems(problems))
		apiGroup.PUT("/submit",
			handlers.PutSubmit(ctx, db, rdb, mc, problems, testcases))
		apiGroup.GET("/stdout", handlers.GetOutput(ctx, db, mc, "stdout"))
		apiGroup.GET("/results", handlers.GetResults(ctx, db, rdb))
		apiGroup.GET("/scores", handlers.GetScores(ctx, rdb, problems, false))
		apiGroup.GET("/me", handlers.GetMe(ctx))

		apiAdminGroup := apiGroup.NewGroup("/admin").
			Use(middlewares.CheckAdmin(ctx))
		apiAdminGroup.GET("/stderr", handlers.GetOutput(ctx, db, mc, "stderr"))
		apiAdminGroup.GET("/scores", handlers.GetScores(ctx, rdb, problems, true))
		apiAdminGroup.POST("/reloadscores",
			handlers.PostReloadScores(ctx, db, rdb, problems))

		// Initialize scheduler router
		scheduler := bunrouter.New()
		scheduleGroup := scheduler.NewGroup("/schedule").
			Use(middlewares.ErrorHandler)
		scheduleGroup.GET("/job", handlers.GetScheduleJob(ctx, rdb))
		scheduleGroup.PUT("/result",
			handlers.PutScheduleResult(ctx, db, rdb, mc, problems, testcases))

		go func() {
			log.Println(http.ListenAndServe(
				":"+viper.GetString("scheduler-port"), scheduler))
		}()
		log.Println(http.ListenAndServe(
			":"+viper.GetString("backend-port"), backend))
	},
}

func init() {
	backendCmd.PersistentFlags().
		String("backend-port", "8080", "Port for backend to listen on")
	viper.BindPFlag("backend-port",
		backendCmd.PersistentFlags().Lookup("backend-port"))
	backendCmd.PersistentFlags().
		String("scheduler-port", "8081", "Port for scheduler to listen on")
	viper.BindPFlag("scheduler-port",
		backendCmd.PersistentFlags().Lookup("scheduler-port"))
	backendCmd.PersistentFlags().
		String("problems-dir", "./problems", "Directory to read problems")
	viper.BindPFlag("problems-dir",
		backendCmd.PersistentFlags().Lookup("problems-dir"))
	backendCmd.PersistentFlags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
	viper.BindPFlag("testcases-dir",
		backendCmd.PersistentFlags().Lookup("testcases-dir"))
	backendCmd.PersistentFlags().
		String("jwt-secret", "poop", "JWT HMAC secret")
	viper.BindPFlag("jwt-secret",
		backendCmd.PersistentFlags().Lookup("jwt-secret"))
	backendCmd.PersistentFlags().
		String("preshared-key", "poop", "Pre-shared key for worker to auth")
	viper.BindPFlag("preshared-key",
		backendCmd.PersistentFlags().Lookup("preshared-key"))
	backendCmd.PersistentFlags().
		String("oauth-client-id", "", "OAuth client id")
	viper.BindPFlag("oauth-client-id",
		backendCmd.PersistentFlags().Lookup("oauth-client-id"))
	backendCmd.PersistentFlags().
		String("oauth-client-secret", "", "OAuth client secret")
	viper.BindPFlag("oauth-client-secret",
		backendCmd.PersistentFlags().Lookup("oauth-client-secret"))
	backendCmd.PersistentFlags().
		String("oauth-redirect-url", "http://localhost:8080/auth/callback",
			"OAuth redirect url")
	viper.BindPFlag("oauth-redirect-url",
		backendCmd.PersistentFlags().Lookup("oauth-redirect-url"))
	backendCmd.PersistentFlags().
		String("redis-host", "localhost:6379", "Redis host[:port]")
	viper.BindPFlag("redis-host",
		backendCmd.PersistentFlags().Lookup("redis-host"))
	backendCmd.PersistentFlags().
		String("redis-user", "nasa", "Redis user")
	viper.BindPFlag("redis-user",
		backendCmd.PersistentFlags().Lookup("redis-user"))
	backendCmd.PersistentFlags().
		String("redis-pass", "", "Redis password")
	viper.BindPFlag("redis-pass",
		backendCmd.PersistentFlags().Lookup("redis-pass"))
	backendCmd.PersistentFlags().
		Int("redis-db", 0, "Redis db")
	viper.BindPFlag("redis-db",
		backendCmd.PersistentFlags().Lookup("redis-db"))
	backendCmd.PersistentFlags().
		String("minio-host", "localhost:9000", "MinIO host[:port]")
	viper.BindPFlag("minio-host",
		backendCmd.PersistentFlags().Lookup("minio-host"))
	backendCmd.PersistentFlags().
		String("minio-bucket", "nasaoj", "MinIO bucket")
	viper.BindPFlag("minio-bucket",
		backendCmd.PersistentFlags().Lookup("minio-bucket"))
	backendCmd.PersistentFlags().
		String("minio-accesskey", "poop", "MinIO accesskey")
	viper.BindPFlag("minio-accesskey",
		backendCmd.PersistentFlags().Lookup("minio-accesskey"))
	backendCmd.PersistentFlags().
		String("minio-secretkey", "poop", "MinIO secretkey")
	viper.BindPFlag("minio-secretkey",
		backendCmd.PersistentFlags().Lookup("minio-secretkey"))
	backendCmd.PersistentFlags().
		Bool("minio-usessl", false, "MinIO usessl")
	viper.BindPFlag("minio-usessl",
		backendCmd.PersistentFlags().Lookup("minio-usessl"))
	backendCmd.PersistentFlags().
		String("admin-role", "nasata", "Admin role in OIDC groups")
	viper.BindPFlag("admin-role",
		backendCmd.PersistentFlags().Lookup("admin-role"))
	backendCmd.PersistentFlags().
		String("cors-origin", "", "CORS origin")
	viper.BindPFlag("cors-origin",
		backendCmd.PersistentFlags().Lookup("cors-origin"))
}
