package serve

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/go-redis/cache/v9"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bunotel"
	"github.com/uptrace/bunrouter"
	apiHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	authHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/auth"
	scheduleHandlers "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/schedule"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	_ "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/docs"
)

//	@title			NASAOJ v3
//	@version		2.0
//	@termsOfService	http://swagger.io/terms/

// @license.name	0BSD
// @BasePath		/
var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "NASAOJ backend server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Initialize Tracing
		appname := "nasaoj-backend"
		tracing.InitTracer(appname)
		if viper.GetString("uptrace-dsn") != "" {
			tracing.InitUptrace(appname)
			defer tracing.DeferUptrace(ctx)
		}

		// Initialize DB instance
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
		bunDB := bun.NewDB(sqldb, pgdialect.New())
		bunDB.AddQueryHook(bunotel.NewQueryHook(bunotel.WithDBName("backend")))

		// Initialize Redis instance
		rdb := redis.NewClient(&redis.Options{
			Addr:     viper.GetString("redis-host"),
			Username: viper.GetString("redis-user"),
			Password: viper.GetString("redis-pass"),
			DB:       viper.GetInt("redis-db"),
		})
		if err := redisotel.InstrumentTracing(rdb); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to instrument tracing on redisotel",
					zap.Error(err))
			panic(err)
		}

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
			tracing.Logger.Ctx(ctx).
				Error("failed to create minio client",
					zap.Error(err))
			panic(err)
		}

		if err := utils.InitMinIO(ctx, mc); err != nil {
			tracing.Logger.Ctx(ctx).
				Error("failed to minio init",
					zap.Error(err))
			panic(err)
		}

		// Initialize custom interfaces
		db := implements.NewBunDatabase(bunDB, implements.DatabaseTypePostgresql)
		s3 := implements.NewMinIOObjectStorage(mc)
		jobQue := implements.NewRedisJobQueue(rdb)
		tokens := implements.NewRedisTokenStorage(rdb)
		workers := implements.NewRedisWorkerStatusStorage(rdb)
		problems := implements.NewDBProblemStorage(bunDB,
			&cache.Options{Redis: rdb})
		tools := implements.NewDBToolStorage(bunDB,
			&cache.Options{Redis: rdb})
		contents := implements.NewDBContentFileStorage(bunDB,
			&cache.Options{Redis: rdb})

		if viper.GetString("admin-token") != "" {
			if err := tokens.SetAdminToken(
				ctx, viper.GetString("admin-token")); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to set admin token",
						zap.Error(err))
				panic(err)
			}
		}

		// Initialize backend router
		router := bunrouter.New()
		backend := router.NewGroup("").
			Use(middlewares.ErrorHandler).
			Use(middlewares.BunrouterOtel).
			Use(middlewares.AccessLog).
			Use(middlewares.CORSHandler)

		if viper.GetBool("swagger") {
			backend.GET("/swagger/*any",
				bunrouter.HTTPHandlerFunc(
					httpSwagger.Handler()))
		}

		authGroup := backend.NewGroup("/auth").
			Use(middlewares.CacheControlHandler)
		authGroup.GET("/login",
			authHandlers.GetLogin(ctx))
		authGroup.PUT("/logout",
			authHandlers.PutLogout(db))
		authGroup.GET("/callback",
			authHandlers.GetCallback(db))

		apiGroup := backend.NewGroup("/api").
			Use(middlewares.CheckAuth(db, tokens))

		apiReadGroup := apiGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanAPIRead: models.ScopeSelf,
			}))
		apiReadGroup.GET("/problem",
			apiHandlers.GetProblem(problems))
		apiReadGroup.GET("/problems",
			apiHandlers.GetProblems(problems, false))
		apiReadGroup.GET("/tool",
			apiHandlers.GetTool(tools))
		apiReadGroup.GET("/tools",
			apiHandlers.GetTools(tools, false))
		apiReadGroup.GET("/stdout",
			apiHandlers.GetOutput(db, mc, "stdout", false))
		apiReadGroup.GET("/results",
			apiHandlers.GetResults(db, problems, false))
		apiReadGroup.GET("/scores",
			apiHandlers.GetScores(db, false))
		apiReadGroup.GET("/me",
			apiHandlers.GetMe)
		apiReadGroup.GET("/submissions",
			apiHandlers.GetSubmissions(db, problems, false))

		apiWriteGroup := apiGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanAPIWrite: models.ScopeSelf,
			}))
		apiWriteGroup.POST("/submit",
			apiHandlers.PostSubmit(
				db, jobQue, s3, problems, tools, contents, tokens, false))

		apiUserArgsGroup := apiGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanUserArgsWrite: models.ScopeSelf,
			}))
		apiUserArgsGroup.PUT("/user",
			apiHandlers.PutUser(db))

		apiAdminGroup := apiGroup.NewGroup("/admin")
		apiAdminReadGroup := apiAdminGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanAPIRead: models.ScopeAll,
			}))
		apiAdminReadGroup.GET("/stdout",
			apiHandlers.GetOutput(db, mc, "stdout", true))
		apiAdminReadGroup.GET("/stderr",
			apiHandlers.GetOutput(db, mc, "stderr", true))
		apiAdminReadGroup.GET("/scores",
			apiHandlers.GetScores(db, true))
		apiAdminReadGroup.GET("/dump-scores/:problemId",
			apiHandlers.GetDumpScores(db))
		apiAdminReadGroup.GET("/user",
			apiHandlers.GetUser(db))
		apiAdminReadGroup.GET("/users",
			apiHandlers.GetUsers(db))
		apiAdminReadGroup.GET("/submissions",
			apiHandlers.GetSubmissions(db, problems, true))
		apiAdminReadGroup.GET("/results",
			apiHandlers.GetResults(db, problems, true))
		apiAdminReadGroup.GET("/problems",
			apiHandlers.GetProblems(problems, true))
		apiAdminReadGroup.GET("/tools",
			apiHandlers.GetTools(tools, true))
		apiAdminReadGroup.GET("/tokens",
			apiHandlers.GetTokens(tokens))
		apiAdminReadGroup.GET("/worker/states",
			apiHandlers.GetWorkerStates(workers))

		apiAdminWriteGroup := apiAdminGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanAPIWrite: models.ScopeAll,
			}))
		apiAdminWriteGroup.PUT("/problems",
			apiHandlers.PutProblems(problems))
		apiAdminWriteGroup.PUT("/content-files",
			apiHandlers.PutContentFiles(contents))
		apiAdminWriteGroup.PUT("/problem/state",
			apiHandlers.PutProblemState(problems))
		apiAdminWriteGroup.PUT("/tools",
			apiHandlers.PutTools(tools))
		apiAdminWriteGroup.PUT("/tool/state",
			apiHandlers.PutToolState(tools))
		apiAdminWriteGroup.PUT("/users",
			apiHandlers.PutUsers(db))
		apiAdminWriteGroup.PUT("/token/state",
			apiHandlers.PutTokenState(tokens))
		apiAdminWriteGroup.POST("/token",
			apiHandlers.PostToken(tokens))
		apiAdminWriteGroup.POST("/submit",
			apiHandlers.PostSubmit(
				db, jobQue, s3, problems, tools, contents, tokens, true))
		apiAdminWriteGroup.DELETE("/problem",
			apiHandlers.DeleteProblem(problems))
		apiAdminWriteGroup.DELETE("/tool",
			apiHandlers.DeleteTool(tools))
		apiAdminWriteGroup.DELETE("/submission",
			apiHandlers.DeleteSubmission(db))
		apiAdminWriteGroup.DELETE("/user",
			apiHandlers.DeleteUser(db))

		scheduleGroup := backend.NewGroup("/schedule").
			Use(middlewares.CheckAuth(db, tokens)).
			Use(middlewares.CheckPrivs(models.Privileges{
				CanScheduleAPI: models.ScopeAll,
			}))
		scheduleGroup.GET("/job",
			scheduleHandlers.GetScheduleJob(db, jobQue, workers, s3))
		scheduleGroup.PUT("/result",
			scheduleHandlers.PutScheduleResult(
				db, jobQue, workers, s3, problems, contents, tokens))

		log.Println(http.ListenAndServe(
			":"+viper.GetString("port"),
			otelhttp.NewHandler(router, "")))
	},
}

func init() {
	// HTTP Server
	backendCmd.Flags().
		String("port", "8080", "Port to listen on")

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
		String("web-oauth-redirect-url", "http://127.0.0.1:8080/auth/callback",
			"OAuth redirect url")
	backendCmd.Flags().
		String("cli-oauth-provider",
			"https://oauth.cs.nycu.edu.tw", "OAuth provider")
	backendCmd.Flags().
		String("cli-oauth-client-id", "", "OAuth client id")
	backendCmd.Flags().
		String("cli-oauth-client-secret", "", "OAuth client secret")
	backendCmd.Flags().
		String("cli-oauth-redirect-url", "http://127.0.0.1:8389/auth/callback",
			"OAuth redirect url")

	// Postgresql
	backendCmd.Flags().
		String("pg-connection-string",
			"postgres://poop:poop@pg:5432/poop?sslmode=disable",
			"Postgres connection string")

	// Redis
	backendCmd.Flags().
		String("redis-host", "127.0.0.1:6379", "Redis host[:port]")
	backendCmd.Flags().
		String("redis-user", "nasa", "Redis user")
	backendCmd.Flags().
		String("redis-pass", "", "Redis password")
	backendCmd.Flags().
		Int("redis-db", 0, "Redis db")

	// MinIO
	backendCmd.Flags().
		String("minio-host", "127.0.0.1:9000", "MinIO host[:port]")
	backendCmd.Flags().
		String("minio-bucket", "nasaoj", "MinIO bucket")
	backendCmd.Flags().
		String("minio-accesskey", "poop", "MinIO accesskey")
	backendCmd.Flags().
		String("minio-secretkey", "poop", "MinIO secretkey")
	backendCmd.Flags().
		Bool("minio-usessl", false, "MinIO usessl")

	// Uptrace
	backendCmd.Flags().
		String("uptrace-dsn", "", "Uptrace DSN (Disabled by default)")

	// Others
	backendCmd.Flags().
		String("admin-token", "", "Bootstrap with an predefined admin token")
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
