package serve

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	httpSwagger "github.com/swaggo/http-swagger"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/extra/bundebug"
	"github.com/uptrace/bun/extra/bunotel"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/crons"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/auth"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/schedule"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/cache"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/implements"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/interfaces"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.uber.org/zap"

	_ "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/docs"
)

//	@title		NASAOJ v3
//	@version	3.0

// @license.name	0BSD
// @BasePath		/
var backendCmd = &cobra.Command{
	Use:   "backend",
	Short: "NASAOJ backend server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		// Initialize Tracing
		appname := "nasaoj-backend"
		if viper.GetString("uptrace-dsn") != "" {
			tracing.InitUptrace(appname, "httpserver:"+appname)
			defer tracing.DeferUptrace(ctx)
		}
		tracing.InitTracer(appname)
		defer tracing.DeferLogger()

		// Initialize DB instance
		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(
			viper.GetString("pg-connection-string"))))
		bunDB := bun.NewDB(sqldb, pgdialect.New())
		bunDB.AddQueryHook(bundebug.NewQueryHook())
		bunDB.AddQueryHook(bunotel.NewQueryHook(bunotel.WithDBName("backend")))

		// Initialize Redis instance, unless running --single-node. Resource-
		// group locks and worker status/lease bookkeeping need to be shared
		// across backend/worker processes, which only Redis can do; a
		// single-node deployment instead uses an in-process store
		// (implements.Local) for those, and Postgres-only caching.
		singleNode := viper.GetBool("single-node")

		var cacheOpt cache.Options
		var jobQueue interfaces.JobQueue
		var workerStore interfaces.WorkerStatusStorage

		if singleNode {
			local := implements.NewLocal()
			jobQueue = local
			workerStore = local
			cacheOpt.LocalCache = cache.NewTinyLFU(10000, 5*time.Minute)
		} else {
			rdb := redis.NewClient(&redis.Options{
				Addr:     viper.GetString("redis-host"),
				Username: viper.GetString("redis-user"),
				Password: viper.GetString("redis-pass"),
				DB:       viper.GetInt("redis-db"),
				MaintNotificationsConfig: &maintnotifications.Config{
					Mode: maintnotifications.ModeDisabled,
				},
			})
			if err := redisotel.InstrumentTracing(rdb); err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to instrument tracing on redisotel",
						zap.Error(err))
				panic(err)
			}

			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			pingErr := rdb.Ping(pingCtx).Err()
			cancel()
			if pingErr != nil {
				tracing.Logger.Ctx(ctx).
					Panic("redis is required for multi-node deployments "+
						"(resource-group locks and worker-status coordination "+
						"need to be shared across backend/worker processes); "+
						"pass --single-node (or set SINGLE_NODE=true) to run "+
						"a single backend without Redis",
						zap.Error(pingErr))
				panic(pingErr)
			}

			kv := implements.NewRedis(rdb)
			jobQueue = kv
			workerStore = kv
			cacheOpt.Redis = rdb
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
		db := implements.NewPostgres(bunDB, &cacheOpt)
		s3 := implements.NewMinIO(mc)

		// Bootstrapping the admin token now writes to Postgres (the token
		// table), which may not exist yet if migrations haven't run - e.g. a
		// fresh deployment where this container starts before `judge migrate
		// up` does. Retry in the background instead of blocking startup /
		// crash-looping the process on it.
		if adminToken := viper.GetString("admin-token"); adminToken != "" {
			go func() {
				for {
					err := db.SetAdminToken(ctx, adminToken)
					if err == nil {
						return
					}
					tracing.Logger.Ctx(ctx).
						Warn("failed to set admin token, retrying",
							zap.Error(err))

					select {
					case <-ctx.Done():
						return
					case <-time.After(time.Second):
					}
				}
			}()
		}

		// Initialize backend router
		router := bunrouter.New()
		backend := router.NewGroup("").
			Use(middlewares.ErrorHandler).
			Use(middlewares.BunrouterOtel).
			Use(middlewares.AccessLog).
			Use(middlewares.CORSHandler)

		backend.GET("/",
			utils.Error404)

		if viper.GetBool("swagger") {
			backend.GET("/swagger/*any",
				bunrouter.HTTPHandlerFunc(
					httpSwagger.Handler(httpSwagger.InstanceName("backend"))))
		}

		authHandlers := auth.NewHandlers(db)
		authGroup := backend.NewGroup("/auth").
			Use(middlewares.CacheControlHandler)
		authGroup.POST("/login/cssso",
			authHandlers.PostLoginCSSSO)
		authGroup.POST("/login/plain",
			authHandlers.PostLoginPlain)
		authGroup.PUT("/logout",
			middlewares.CheckAuth(db, db)(
				authHandlers.PutLogout))
		authGroup.GET("/callback",
			authHandlers.GetCallback)

		apiHandlers := api.NewHandlers(db, workerStore, s3)
		backend.GET("/api/healthz",
			apiHandlers.GetHealthz)
		apiGroup := backend.NewGroup("/api").
			Use(middlewares.CheckAuth(db, db))

		apiReadGroup := apiGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanAPIRead: models.ScopeSelf,
			})).
			Use(middlewares.CheckNamespace(db, models.NamespaceRead))
		apiReadGroup.GET("/scores/sum",
			apiHandlers.GetScoreSum)
		apiReadGroup.GET("/problem",
			apiHandlers.GetProblem)
		apiReadGroup.GET("/problems",
			apiHandlers.GetProblems(false))
		apiReadGroup.GET("/tool",
			apiHandlers.GetTool)
		apiReadGroup.GET("/tools",
			apiHandlers.GetTools(false))
		apiReadGroup.GET("/stdout",
			apiHandlers.GetOutput("stdout", false))
		apiReadGroup.GET("/results",
			apiHandlers.GetResults(false))
		apiReadGroup.GET("/scores",
			apiHandlers.GetScores(false))
		apiReadGroup.GET("/submissions",
			apiHandlers.GetSubmissions(false))

		// /me is identity, not namespace-scoped data: only require auth/read.
		apiSelfGroup := apiGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanAPIRead: models.ScopeSelf,
			}))
		apiSelfGroup.GET("/me",
			apiHandlers.GetMe)

		apiWriteGroup := apiGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanAPIWrite: models.ScopeSelf,
			})).
			Use(middlewares.CheckNamespace(db, models.NamespaceRead))
		apiWriteGroup.POST("/submit",
			apiHandlers.PostSubmit(false))
		apiWriteGroup.PUT("/submission/cancel",
			apiHandlers.PutCancelSubmission(false))

		apiJobGroup := apiGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanUserArgsWrite: models.ScopeSelf,
			}))
		apiJobGroup.PUT("/user",
			apiHandlers.PutUser)
		apiJobGroup.PUT("/score",
			apiHandlers.PutScore)

		apiAdminGroup := apiGroup.NewGroup("/admin")
		apiAdminReadGroup := apiAdminGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanAPIRead: models.ScopeAll,
			})).
			Use(middlewares.CheckNamespace(db, models.NamespaceAdmin))
		apiAdminReadGroup.GET("/namespaces",
			apiHandlers.GetNamespaces)
		apiAdminReadGroup.GET("/stdout",
			apiHandlers.GetOutput("stdout", true))
		apiAdminReadGroup.GET("/stderr",
			apiHandlers.GetOutput("stderr", true))
		apiAdminReadGroup.GET("/scores",
			apiHandlers.GetScores(true))
		apiAdminReadGroup.GET("/dump-scores/:problemId",
			apiHandlers.GetDumpScores)
		apiAdminReadGroup.GET("/user",
			apiHandlers.GetUser)
		apiAdminReadGroup.GET("/users",
			apiHandlers.GetUsers)
		apiAdminReadGroup.GET("/submissions",
			apiHandlers.GetSubmissions(true))
		apiAdminReadGroup.GET("/results",
			apiHandlers.GetResults(true))
		apiAdminReadGroup.GET("/problems",
			apiHandlers.GetProblems(true))
		apiAdminReadGroup.GET("/tools",
			apiHandlers.GetTools(true))
		apiAdminReadGroup.GET("/tokens",
			apiHandlers.GetTokens)
		apiAdminReadGroup.GET("/worker/states",
			apiHandlers.GetWorkerStates)

		apiAdminWriteGroup := apiAdminGroup.NewGroup("").
			Use(middlewares.CheckPrivs(models.Privileges{
				CanAPIWrite: models.ScopeAll,
			})).
			Use(middlewares.CheckNamespace(db, models.NamespaceAdmin))
		apiAdminWriteGroup.POST("/namespace",
			apiHandlers.PostNamespace)
		apiAdminWriteGroup.PUT("/namespace/state",
			apiHandlers.PutNamespaceState)
		apiAdminWriteGroup.PUT("/problems",
			apiHandlers.PutProblems)
		apiAdminWriteGroup.PUT("/content-files",
			apiHandlers.PutContentFiles)
		apiAdminWriteGroup.PUT("/problem/state",
			apiHandlers.PutProblemState)
		apiAdminWriteGroup.PUT("/tools",
			apiHandlers.PutTools)
		apiAdminWriteGroup.PUT("/tool/state",
			apiHandlers.PutToolState)
		apiAdminWriteGroup.PUT("/users/cssso",
			apiHandlers.PutUsersCSSSO)
		apiAdminWriteGroup.PUT("/users/plain",
			apiHandlers.PutUsersPlain)
		apiAdminWriteGroup.PUT("/token/state",
			apiHandlers.PutTokenState)
		apiAdminWriteGroup.POST("/token",
			apiHandlers.PostToken)
		apiAdminWriteGroup.POST("/submit",
			apiHandlers.PostSubmit(true))
		apiAdminWriteGroup.DELETE("/problem",
			apiHandlers.DeleteProblem)
		apiAdminWriteGroup.DELETE("/tool",
			apiHandlers.DeleteTool)
		apiAdminWriteGroup.DELETE("/submission",
			apiHandlers.DeleteSubmission)
		apiAdminWriteGroup.DELETE("/user",
			apiHandlers.DeleteUser)
		apiAdminWriteGroup.PUT("/submission/cancel",
			apiHandlers.PutCancelSubmission(true))

		// Namespace member management is allowed for namespace admins (AdminIn)
		// as well as global admins; CheckNamespace(NamespaceAdmin) enforces both.
		apiNsAdminGroup := apiAdminGroup.NewGroup("").
			Use(middlewares.CheckNamespace(db, models.NamespaceAdmin))
		apiNsAdminGroup.PUT("/namespace/users",
			apiHandlers.PutNamespaceUsers)
		apiNsAdminGroup.DELETE("/namespace/users",
			apiHandlers.DeleteNamespaceUsers)
		apiNsAdminGroup.GET("/namespace/users",
			apiHandlers.GetNamespaceUsers)

		scheduleHandlers := schedule.NewHandlers(db, jobQueue, workerStore, s3)
		scheduleGroup := backend.NewGroup("/schedule").
			Use(middlewares.CheckAuth(db, db)).
			Use(middlewares.CheckPrivs(models.Privileges{
				CanScheduleAPI: models.ScopeAll,
			}))
		scheduleGroup.GET("/submission",
			scheduleHandlers.GetScheduleSubmission)
		scheduleGroup.POST("/job/marshal",
			scheduleHandlers.PostScheduleMarshalJob)
		scheduleGroup.PUT("/result",
			scheduleHandlers.PutScheduleResult)
		scheduleGroup.POST("/heartbeat",
			scheduleHandlers.PostScheduleHeartbeat)

		// Start cron jobs
		go crons.CheckWorkerStatus(ctx, workerStore, db)
		// Submission-level lease reclaim supersedes per-job orphan cleanup.
		go crons.ReclaimSubmissions(ctx, db, db)

		log.Println(http.ListenAndServe(
			":"+viper.GetString("port"),
			otelhttp.NewHandler(router, "")))
	},
}

func init() {
	// HTTP Server
	backendCmd.Flags().
		String("port", "8080", "Port to listen on")
	backendCmd.Flags().
		String("namespace-source", "header",
			"Where request namespaces come from: 'header' (X-Namespace) or "+
				"'jwt' (derive from the caller's membership)")

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
		Bool("single-node", false,
			"Run without Redis: resource-group locks and worker status "+
				"become process-local, and caching is process-local instead "+
				"of shared. Only correct with exactly one backend process; "+
				"required when Redis isn't configured (env SINGLE_NODE).")
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

	// Logging
	backendCmd.Flags().
		String("log-level", "", "Log level (debug, info, warn, error)")

	// Others
	backendCmd.Flags().
		String("admin-token", "", "Bootstrap with an predefined admin token")
	backendCmd.Flags().
		String("admin-role", "", "Admin role in OIDC groups")
	backendCmd.Flags().
		Bool("allow-user-register", false,
			"Allow user to register new UserResource")
	backendCmd.Flags().
		String("cors-origin", "", "CORS origin")
	backendCmd.Flags().
		Bool("swagger", false, "Enable swagger")
	backendCmd.Flags().
		Int64("worker-timeout", 300, "Worker timeout (in sec)")
}
