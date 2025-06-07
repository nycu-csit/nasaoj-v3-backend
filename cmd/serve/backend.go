package serve

import (
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
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
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
		ctx := cmd.Context()

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
		bunDB := bun.NewDB(sqldb, pgdialect.New())

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

		// Initialize custom interfaces
		db := implements.NewBunDatabase(bunDB)
		s3 := implements.NewMinIOObjectStorage(mc)
		jobQue := implements.NewRedisJobQueue(rdb)

		// Initialize backend router
		backend := bunrouter.New()

		backend.GET("/swagger/*any",
			bunrouter.HTTPHandlerFunc(
				httpSwagger.Handler()))

		authGroup := backend.NewGroup("/auth").
			Use(middlewares.ErrorHandler).
			Use(middlewares.CORSHandler)
		authGroup.GET("/login", handlers.GetLogin(oauth2Config))
		authGroup.GET("/callback", handlers.GetCallback(ctx, bunDB,
			oauth2Config, provider.Verifier(&oidc.Config{
				ClientID: viper.GetString("oauth-client-id"),
			})))

		apiGroup := backend.NewGroup("/api").
			Use(middlewares.ErrorHandler).
			Use(middlewares.CORSHandler).
			Use(middlewares.CheckJWT(ctx, bunDB))
		apiGroup.GET("/problems", handlers.GetProblems(problems))
		apiGroup.PUT("/submit", handlers.PutSubmit(ctx, db, jobQue,
			s3, problems, testcases, false))
		apiGroup.GET("/stdout", handlers.GetOutput(ctx, bunDB, mc, "stdout", false))
		apiGroup.GET("/results", handlers.GetResults(ctx, db, false))
		apiGroup.GET("/scores",
			handlers.GetScores(ctx, db, false))
		apiGroup.GET("/me", handlers.GetMe(ctx))
		apiGroup.GET("/submissions",
			handlers.GetSubmissions(ctx, bunDB, rdb, false))

		apiAdminGroup := apiGroup.NewGroup("/admin").
			Use(middlewares.CheckAdmin(ctx))
		apiAdminGroup.GET("/stdout",
			handlers.GetOutput(ctx, bunDB, mc, "stdout", true))
		apiAdminGroup.GET("/stderr",
			handlers.GetOutput(ctx, bunDB, mc, "stderr", true))
		apiAdminGroup.GET("/scores",
			handlers.GetScores(ctx, db, true))
		apiAdminGroup.GET("/users", handlers.GetUsers(ctx, bunDB))
		apiAdminGroup.GET("/submissions",
			handlers.GetSubmissions(ctx, bunDB, rdb, true))
		apiAdminGroup.GET("/results", handlers.GetResults(ctx, db, true))
		apiAdminGroup.PUT("/submit",
			handlers.PutSubmit(ctx, db, jobQue, s3, problems, testcases, true))

		// Initialize scheduler router
		scheduler := bunrouter.New()
		scheduleGroup := scheduler.NewGroup("/schedule").
			Use(middlewares.ErrorHandler).
			Use(middlewares.CheckPreSharedKey)
		scheduleGroup.GET("/job", handlers.GetScheduleJob(ctx, db, jobQue))
		scheduleGroup.PUT("/result", handlers.PutScheduleResult(ctx, db,
			jobQue, s3, problems, testcases))

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
	backendCmd.PersistentFlags().
		String("scheduler-port", "8081", "Port for scheduler to listen on")
	backendCmd.PersistentFlags().
		String("problems-dir", "./problems", "Directory to read problems")
	backendCmd.PersistentFlags().
		String("testcases-dir", "./testcases", "Directory to read testcases")
	backendCmd.PersistentFlags().
		String("jwt-secret", "poop", "JWT HMAC secret")
	backendCmd.PersistentFlags().
		String("preshared-key", "poop", "Pre-shared key for worker to auth")
	backendCmd.PersistentFlags().
		String("oauth-client-id", "", "OAuth client id")
	backendCmd.PersistentFlags().
		String("oauth-client-secret", "", "OAuth client secret")
	backendCmd.PersistentFlags().
		String("oauth-redirect-url", "http://localhost:8080/auth/callback",
			"OAuth redirect url")
	backendCmd.PersistentFlags().
		String("pg-connection-string",
			"postgres://poop:poop@pg:5432/poop?sslmode=disable",
			"Postgres connection string")
	backendCmd.PersistentFlags().
		String("redis-host", "localhost:6379", "Redis host[:port]")
	backendCmd.PersistentFlags().
		String("redis-user", "nasa", "Redis user")
	backendCmd.PersistentFlags().
		String("redis-pass", "", "Redis password")
	backendCmd.PersistentFlags().
		Int("redis-db", 0, "Redis db")
	backendCmd.PersistentFlags().
		String("minio-host", "localhost:9000", "MinIO host[:port]")
	backendCmd.PersistentFlags().
		String("minio-bucket", "nasaoj", "MinIO bucket")
	backendCmd.PersistentFlags().
		String("minio-accesskey", "poop", "MinIO accesskey")
	backendCmd.PersistentFlags().
		String("minio-secretkey", "poop", "MinIO secretkey")
	backendCmd.PersistentFlags().
		Bool("minio-usessl", false, "MinIO usessl")
	backendCmd.PersistentFlags().
		String("admin-role", "nasata", "Admin role in OIDC groups")
	backendCmd.PersistentFlags().
		String("cors-origin", "", "CORS origin")
}
