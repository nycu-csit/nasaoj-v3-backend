package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bunrouter"
	"golang.org/x/oauth2"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/cmd"
	_ "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/docs"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/env"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
)

var rootCmd = &cobra.Command{
	Use:   "nasaoj",
	Short: "NASAOJ backend server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()

		provider, err := oidc.NewProvider(ctx, "https://oauth.cs.nycu.edu.tw")
		if err != nil {
			panic(err)
		}

		// Configure an OpenID Connect aware OAuth2 client.
		oauth2Config := oauth2.Config{
			ClientID:     env.OAUTH_CLIENT_ID,
			ClientSecret: env.OAUTH_CLIENT_SECRET,
			RedirectURL:  env.OAUTH_REDIRECT_URL,

			// Discovery returns the OAuth2 endpoints.
			Endpoint: provider.Endpoint(),

			// "openid" is a required scope for OpenID Connect flows.
			Scopes: []string{oidc.ScopeOpenID, "csid", "groups"},
		}

		// Initialize DB instance
		dsn := env.PG_CONNECTION_STRING

		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		// Initialize Redis instance
		rdb := redis.NewClient(&redis.Options{
			Addr:     env.REDIS_HOST,
			Username: env.REDIS_USER,
			Password: env.REDIS_PASS,
			DB:       env.REDIS_DB,
		})

		// Initialize MinIO instance
		mc, err := minio.New(env.MINIO_HOST, &minio.Options{
			Creds: credentials.NewStaticV4(
				env.MINIO_ACCESSKEY,
				env.MINIO_SECRETKEY,
				"",
			),
			Secure: env.MINIO_USESSL,
		})
		if err != nil {
			panic(err)
		}

		if err := utils.InitMinIO(ctx, mc); err != nil {
			panic(err)
		}

		// Load problems and testcases
		problems, err := utils.LoadProblems(os.Getenv("PROBLEMS_DIR"))
		if err != nil {
			panic(err)
		}
		testcases, err := utils.LoadTestcases(os.Getenv("TESTCASES_DIR"))
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
				ClientID: os.Getenv("OAUTH_CLIENT_ID"),
			})))

		apiGroup := backend.NewGroup("/api").
			Use(middlewares.ErrorHandler).
			Use(middlewares.CheckJWT(ctx, db))
		apiGroup.GET("/problems", handlers.GetProblems(problems))
		apiGroup.PUT("/submit",
			handlers.PutSubmit(ctx, db, rdb, mc, problems, testcases))
		apiGroup.GET("/output", handlers.GetOutput(ctx, db, mc))
		apiGroup.GET("/results", handlers.GetResults(ctx, db, rdb))
		apiGroup.GET("/scores", handlers.GetScores(ctx, rdb, problems))
		apiGroup.GET("/me", handlers.GetMe(ctx))

		apiAdminGroup := apiGroup.NewGroup("/admin").
			Use(middlewares.CheckAdmin(ctx))
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
			log.Println(http.ListenAndServe(":"+os.Getenv("SCHEDULER_PORT"), scheduler))
		}()
		log.Println(http.ListenAndServe(":"+os.Getenv("BACKEND_PORT"), backend))
	},
}

//	@title			NASAOJ v3
//	@version		2.0
//	@termsOfService	http://swagger.io/terms/

//	@license.name	WTFPL
//	@BasePath		/

// @Securitydefinitions.apikey	ApiKeyAuth
// @In							header
// @Name						Cookie
// @Description				jwt=value, get it at /auth/login
func main() {
	env.InitEnv()

	rootCmd.AddCommand(cmd.InitDBCmd)
	rootCmd.AddCommand(cmd.ImportUserCmd)
	rootCmd.Execute()
}
