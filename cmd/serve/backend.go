package serve

import (
	"database/sql"
	"log"
	"net/http"

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
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	authHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/auth"
	scheduleHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/schedule"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"

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

		// Initialize custom interfaces
		db := implements.NewBunDatabase(bunDB)
		s3 := implements.NewMinIOObjectStorage(mc)
		jobQue := implements.NewRedisJobQueue(rdb)
		problems := implements.NewDBProblemStorage(db)

		// Initialize backend router
		backend := bunrouter.New()

		if viper.GetBool("swagger") {
			backend.GET("/swagger/*any",
				bunrouter.HTTPHandlerFunc(
					httpSwagger.Handler()))
		}

		authGroup := backend.NewGroup("/auth").
			Use(middlewares.ErrorHandler).
			Use(middlewares.CORSHandler)
		authGroup.GET("/login", authHandlers.GetLogin(ctx))
		authGroup.PUT("/logout", authHandlers.PutLogout(db))
		authGroup.GET("/callback", authHandlers.GetCallback(db))

		apiGroup := backend.NewGroup("/api").
			Use(middlewares.ErrorHandler).
			Use(middlewares.CORSHandler).
			Use(middlewares.CheckRefreshToken(db)).
			Use(middlewares.CheckAccessToken(db))
		apiGroup.GET("/problems", apiHandlers.GetProblems(db, false))
		apiGroup.POST("/submit", apiHandlers.PostSubmit(db, jobQue,
			s3, problems, false))
		apiGroup.GET("/stdout", apiHandlers.GetOutput(db, mc, "stdout", false))
		apiGroup.GET("/results", apiHandlers.GetResults(db, false))
		apiGroup.GET("/scores",
			apiHandlers.GetScores(db, false))
		apiGroup.GET("/me", apiHandlers.GetMe)
		apiGroup.GET("/submissions", apiHandlers.GetSubmissions(db, false))

		apiAdminGroup := apiGroup.NewGroup("/admin").
			Use(middlewares.CheckAdmin)
		apiAdminGroup.GET("/stdout",
			apiHandlers.GetOutput(db, mc, "stdout", true))
		apiAdminGroup.GET("/stderr",
			apiHandlers.GetOutput(db, mc, "stderr", true))
		apiAdminGroup.GET("/scores",
			apiHandlers.GetScores(db, true))
		apiAdminGroup.GET("/user", apiHandlers.GetUser(db))
		apiAdminGroup.GET("/users", apiHandlers.GetUsers(db))
		apiAdminGroup.GET("/submissions", apiHandlers.GetSubmissions(db, true))
		apiAdminGroup.GET("/results", apiHandlers.GetResults(db, true))
		apiAdminGroup.POST("/submit",
			apiHandlers.PostSubmit(db, jobQue, s3, problems, true))
		apiAdminGroup.GET("/problems", apiHandlers.GetProblems(db, true))
		apiAdminGroup.PUT("/problems", apiHandlers.PutProblems(problems))
		apiAdminGroup.PUT("/problem-state", apiHandlers.PutProblemState(problems))
		apiAdminGroup.PUT("/users", apiHandlers.PutUsers(db))

		// Initialize scheduler router
		scheduler := bunrouter.New()
		scheduleGroup := scheduler.NewGroup("/schedule").
			Use(middlewares.ErrorHandler).
			Use(middlewares.CheckPreSharedKey)
		scheduleGroup.GET("/job", scheduleHandlers.GetScheduleJob(db, jobQue))
		scheduleGroup.PUT("/result", scheduleHandlers.PutScheduleResult(db,
			jobQue, s3, problems))

		go func() {
			log.Println(http.ListenAndServe(
				":"+viper.GetString("scheduler-port"), scheduler))
		}()
		log.Println(http.ListenAndServe(
			":"+viper.GetString("backend-port"), backend))
	},
}

func init() {
	// HTTP Server
	backendCmd.Flags().
		String("backend-port", "8080", "Port for backend to listen on")
	backendCmd.Flags().
		String("scheduler-port", "8081", "Port for scheduler to listen on")

	// JWT Cookies
	backendCmd.Flags().
		Int64("access-token-timeout", 300, "Timeout of Access Token JWT")
	backendCmd.Flags().
		String("access-token-secret", "poop", "Access Token JWT HMAC secret")
	backendCmd.Flags().
		Int64("refresh-token-timeout", 3600, "Timeout of Refresh Token JWT")
	backendCmd.Flags().
		String("refresh-token-secret", "poop", "Refresh Token JWT HMAC secret")

	// OAuth2 / OIDC
	backendCmd.Flags().
		String("web-oauth-provider",
			"https://oauth.cs.nycu.edu.tw", "OAuth provider")
	backendCmd.Flags().
		String("web-oauth-client-id", "", "OAuth client id")
	backendCmd.Flags().
		String("web-oauth-client-secret", "", "OAuth client secret")
	backendCmd.Flags().
		String("web-oauth-redirect-url", "http://localhost:8080/auth/callback",
			"OAuth redirect url")
	backendCmd.Flags().
		String("cli-oauth-provider",
			"https://oauth.cs.nycu.edu.tw", "OAuth provider")
	backendCmd.Flags().
		String("cli-oauth-client-id", "", "OAuth client id")
	backendCmd.Flags().
		String("cli-oauth-client-secret", "", "OAuth client secret")
	backendCmd.Flags().
		String("cli-oauth-redirect-url", "http://localhost:8389/auth/callback",
			"OAuth redirect url")

	// Scheduler
	backendCmd.Flags().
		String("preshared-key", "poop", "Pre-shared key for worker to auth")

	// Postgresql
	backendCmd.Flags().
		String("pg-connection-string",
			"postgres://poop:poop@pg:5432/poop?sslmode=disable",
			"Postgres connection string")

	// Redis
	backendCmd.Flags().
		String("redis-host", "localhost:6379", "Redis host[:port]")
	backendCmd.Flags().
		String("redis-user", "nasa", "Redis user")
	backendCmd.Flags().
		String("redis-pass", "", "Redis password")
	backendCmd.Flags().
		Int("redis-db", 0, "Redis db")

	// MinIO
	backendCmd.Flags().
		String("minio-host", "localhost:9000", "MinIO host[:port]")
	backendCmd.Flags().
		String("minio-bucket", "nasaoj", "MinIO bucket")
	backendCmd.Flags().
		String("minio-accesskey", "poop", "MinIO accesskey")
	backendCmd.Flags().
		String("minio-secretkey", "poop", "MinIO secretkey")
	backendCmd.Flags().
		Bool("minio-usessl", false, "MinIO usessl")

	// Others
	backendCmd.Flags().
		String("admin-role", "nasata", "Admin role in OIDC groups")
	backendCmd.Flags().
		Bool("allow-user-register", false,
			"Allow user to register new UserResource")
	backendCmd.Flags().
		String("cors-origin", "", "CORS origin")
	backendCmd.Flags().
		Bool("swagger", false, "Enable swagger")
}
