package env

import (
	"os"

	_ "github.com/joho/godotenv/autoload"
)

var (
	ACCESS_TOKEN string

	BACKEND_PORT   string
	SCHEDULER_PORT string

	PROBLEMS_DIR  string
	TESTCASES_DIR string

	JWT_SECRET    string
	PRESHARED_KEY string

	OAUTH_CLIENT_ID     string
	OAUTH_CLIENT_SECRET string
	OAUTH_REDIRECT_URL  string

	PG_CONNECTION_STRING string

	REDIS_HOST string
	REDIS_USER string
	REDIS_PASS string
	REDIS_DB   string

	MINIO_HOST        string
	MINIO_BUCKET      string
	MINIO_ACCESSKEY   string
	MINIO_SECRETKEY   string
	MINIO_PUBLIC_HOST string

	ADMIN_ROLE string
)

func InitEnv() error {
	ACCESS_TOKEN = os.Getenv("ACCESS_TOKEN")

	BACKEND_PORT = os.Getenv("BACKEND_PORT")
	SCHEDULER_PORT = os.Getenv("SCHEDULER_PORT")

	PROBLEMS_DIR = os.Getenv("PROBLEMS_DIR")
	TESTCASES_DIR = os.Getenv("TESTCASES_DIR")

	JWT_SECRET = os.Getenv("JWT_SECRET")
	PRESHARED_KEY = os.Getenv("PRESHARED_KEY")

	OAUTH_CLIENT_ID = os.Getenv("OAUTH_CLIENT_ID")
	OAUTH_CLIENT_SECRET = os.Getenv("OAUTH_CLIENT_SECRET")
	OAUTH_REDIRECT_URL = os.Getenv("OAUTH_REDIRECT_URL")

	PG_CONNECTION_STRING = os.Getenv("PG_CONNECTION_STRING")

	REDIS_HOST = os.Getenv("REDIS_HOST")
	REDIS_USER = os.Getenv("REDIS_USER")
	REDIS_PASS = os.Getenv("REDIS_PASS")
	REDIS_DB = os.Getenv("REDIS_DB")

	MINIO_HOST = os.Getenv("MINIO_HOST")
	MINIO_BUCKET = os.Getenv("MINIO_BUCKET")
	MINIO_ACCESSKEY = os.Getenv("MINIO_ACCESSKEY")
	MINIO_SECRETKEY = os.Getenv("MINIO_SECRETKEY")
	MINIO_PUBLIC_HOST = os.Getenv("MINIO_PUBLIC_HOST")

	ADMIN_ROLE = os.Getenv("ADMIN_ROLE")
	return nil
}
