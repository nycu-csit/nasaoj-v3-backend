package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/utils"
	"golang.org/x/oauth2"

	_ "github.com/joho/godotenv/autoload"
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
func main() {
	ctx := context.Background()

	provider, err := oidc.NewProvider(ctx, "https://oauth.cs.nycu.edu.tw")
	if err != nil {
		panic(err)
	}

	// Configure an OpenID Connect aware OAuth2 client.
	oauth2Config := oauth2.Config{
		ClientID:     os.Getenv("OAUTH_CLIENT_ID"),
		ClientSecret: os.Getenv("OAUTH_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OAUTH_REDIRECT_URL"),

		// Discovery returns the OAuth2 endpoints.
		Endpoint: provider.Endpoint(),

		// "openid" is a required scope for OpenID Connect flows.
		Scopes: []string{oidc.ScopeOpenID, "csid", "groups"},
	}

	// Initialize DB instance
	dsn := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable",
		os.Getenv("PG_USER"), os.Getenv("PG_PASS"),
		os.Getenv("PG_HOST"), os.Getenv("PG_DB"))

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Initialize Redis instance
	redisdb, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		panic(err)
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASS"),
		DB:       redisdb,
	})

	// Initialize MinIO instance
	mc, err := minio.New(os.Getenv("MINIO_HOST"), &minio.Options{
		Creds: credentials.NewStaticV4(
			os.Getenv("MINIO_ACCESSKEY"),
			os.Getenv("MINIO_SECRETKEY"),
			"",
		),
		Secure: false,
	})
	if err != nil {
		panic(err)
	}

	utils.InitDB(ctx, db)
	if err != nil {
		panic(err)
	}

	utils.InitMinIO(ctx, mc)
	if err != nil {
		panic(err)
	}

	// Load problems and testcases
	problems, err := utils.LoadProblems(os.Getenv("PROBLEMS_DIR"))
	if err != nil {
		panic(err)
	}
	testcases, err := utils.LoadTestcases(os.Getenv("TESTCASES_DIR"))

	// Initialize backend router
	backend := bunrouter.New()

	backend.GET("/swagger/*any",
		bunrouter.HTTPHandlerFunc(
			httpSwagger.Handler()))

	authGroup := backend.NewGroup("/auth").
		Use(middlewares.ErrorHandler)
	authGroup.GET("/login", handlers.GetLogin(oauth2Config))
	authGroup.GET("/callback",
		handlers.GetCallback(ctx, db, oauth2Config, provider.Verifier(&oidc.Config{
			ClientID: os.Getenv("OAUTH_CLIENT_ID"),
		})))

	apiGroup := backend.NewGroup("/api").
		Use(middlewares.ErrorHandler).
		Use(middlewares.CheckJWT(ctx, db))
	apiGroup.GET("/problems", handlers.GetProblems(problems))
	apiGroup.PUT("/submit", handlers.PutSubmit(ctx, db, rdb, problems, testcases))
	apiGroup.GET("/output", handlers.GetOutput(ctx, mc))
	apiGroup.GET("/results", handlers.GetResults(ctx, db))
	apiGroup.GET("/scores", handlers.GetScores(ctx, rdb))

	// Initialize scheduler router
	scheduler := bunrouter.New()

	scheduleGroup := scheduler.NewGroup("/schedule").
		Use(middlewares.ErrorHandler)
	scheduleGroup.GET("/job", handlers.GetScheduleJob(ctx, rdb))
	scheduleGroup.PUT("/result", handlers.PutScheduleResult(ctx, db, mc))

	go func() {
		log.Println(http.ListenAndServe(":"+os.Getenv("SCHEDULER_PORT"), scheduler))
	}()
	log.Println(http.ListenAndServe(":"+os.Getenv("BACKEND_PORT"), backend))
}
