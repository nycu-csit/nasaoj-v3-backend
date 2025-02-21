package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"github.com/uptrace/bunrouter"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/middlewares"
	"golang.org/x/oauth2"

	_ "github.com/joho/godotenv/autoload"
	_ "gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/docs"
)

//	@title			Swagger Example API
//	@version		2.0
//	@termsOfService	http://swagger.io/terms/

//	@license.name	WTFPL
//	@BasePath		/

//	@Securitydefinitions.apikey	ApiKeyAuth
//	@In							header
//	@Name						Cookie
//	@Description				jwt=value, get it at /auth/login
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

	/*
		// Initialize DB instance
		dsn := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable",
			os.Getenv("PG_USER"), os.Getenv("PG_PASS"),
			os.Getenv("PG_HOST"), os.Getenv("PG_DB"))

		sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		db := bun.NewDB(sqldb, pgdialect.New())

		// Initialize Redis instance
		redisdb, err := strconv.Atoi(os.Getenv("REDIS_DB"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "REDIS_DB is not integer\n")
			return
		}
		rdb := redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_HOST"),
			Password: os.Getenv("REDIS_PASS"),
			DB:       redisdb,
		})
	*/

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
		fmt.Fprintf(os.Stderr, "MinIO init failed\n")
		return
	}

	// Load problems and stages
	problems, err := LoadFiles()

	// Initialize router
	router := bunrouter.New()

	router.GET("/swagger/*any",
		bunrouter.HTTPHandlerFunc(
			httpSwagger.Handler()))

	authGroup := router.NewGroup("/auth").
		Use(middlewares.ErrorHandler)
	authGroup.GET("/login", handlers.GetLogin(oauth2Config))
	authGroup.GET("/callback",
		handlers.GetCallback(ctx, oauth2Config, provider.Verifier(&oidc.Config{
			ClientID: os.Getenv("OAUTH_CLIENT_ID"),
		})))

	apiGroup := router.NewGroup("/api").
		Use(middlewares.ErrorHandler).
		Use(middlewares.CheckJWT)
	apiGroup.GET("/problems", handlers.GetProblems(problems))
	apiGroup.GET("/outputs", handlers.GetOutput(ctx, mc))
	//router.GET("/api/results", GetResults(ctx, db))
	//router.GET("/api/scores", GetScores(ctx, db))

	//router.PUT("/api/submit", PutSubmit())

	log.Println(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
