package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bunrouter"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	ctx := context.Background()

	// Initialize DB instance
	dsn := fmt.Sprintf("postgres://%s:%s@%s:5432/%s?sslmode=disable",
		os.Getenv("PG_USER"), os.Getenv("PG_PASS"),
		os.Getenv("PG_HOST"), os.Getenv("PG_DB"))

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	db := bun.NewDB(sqldb, pgdialect.New())

	// Initialize Redis instance
	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "REDIS_DB is not integer\n")
		return
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST"),
		Password: os.Getenv("REDIS_PASS"),
		DB:       redisDB,
	})

	// Initialize MinIO instance
	mc, err := minio.New(os.Getenv("MINIO_HOST"), &minio.Options{
		Creds: credentials.NewStaticV4(
			os.Getenv("MINIO_ACCESSKEY"),
			os.Getenv("MINIO_SECRETKEY"),
			""),
		Secure: false,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "MinIO initialization failed\n")
		return
	}

	// Initialize router
	router := bunrouter.New()

	router.GET("/api/job", GetJob(ctx, rdb))
	router.PUT("/api/job", PutJob(ctx, rdb))
	router.PUT("/api/result", PutResult(ctx, db, mc))

	log.Println(http.ListenAndServe(":"+os.Getenv("PORT"), router))
}
