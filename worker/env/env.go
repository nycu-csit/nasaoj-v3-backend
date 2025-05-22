package env

import (
	"os"
	"strconv"

	_ "github.com/joho/godotenv/autoload"
)

var (
	THREADS        int64
	BUFFER_SIZE    int64
	THREAD_BACKOFF int64

	TAGS string

	SCHEDULER_URL string
	PRESHARED_KEY string

	RUNNING_DIR string
)

func InitEnv() error {
	var err error

	THREADS, err = strconv.ParseInt(os.Getenv("THREADS"), 10, 64)
	if err != nil {
		return err
	}
	BUFFER_SIZE, err = strconv.ParseInt(os.Getenv("BUFFER_SIZE"), 10, 64)
	if err != nil {
		return err
	}
	THREAD_BACKOFF, err = strconv.ParseInt(os.Getenv("THREAD_BACKOFF"), 10, 64)
	if err != nil {
		return err
	}
	TAGS = os.Getenv("TAGS")

	SCHEDULER_URL = os.Getenv("SCHEDULER_URL")
	PRESHARED_KEY = os.Getenv("PRESHARED_KEY")

	RUNNING_DIR = os.Getenv("RUNNING_DIR")
	return nil
}
