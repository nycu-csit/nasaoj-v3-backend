package tests

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

var client *resty.Client

func Test_00_Healthz(t *testing.T) {
	for i := 0; i < 10; i++ {
		resp, err := client.R().
			Get("/api/healthz")
		if err == nil && resp.StatusCode() == http.StatusOK {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatal("server did not ready")
}

func setCookie(c *resty.Client, r *resty.Response) {
	for _, value := range r.Header().Values("Set-Cookie") {
		cookie, _ := http.ParseSetCookie(value)
		client.SetCookie(cookie)
	}
}

func init() {
	client = resty.New()
	client.SetBaseURL("http://localhost:8080")
	client.SetRedirectPolicy(resty.NoRedirectPolicy())
	client.OnSuccess(setCookie)
}
