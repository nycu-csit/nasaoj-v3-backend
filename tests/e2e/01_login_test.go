package tests

import (
	"net/http"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/auth"
)

func Test_01_1_Create_User(t *testing.T) {
	t.Logf("access with admin token: %s\n", viper.GetString("admin-token"))

	stdin := strings.NewReader(`---
- username: 'poop'
  password: 'poop'
`)

	cmd := exec.Command(
		"judge", "admin", "import", "users", "plain",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080",
	)
	var out strings.Builder
	cmd.Stdin = stdin
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Logf("finish with %v\n", cmd)
		t.Logf("output: %s\n", out.String())

		t.Fatal("failed to import user")
	}
}

func Test_01_2_Login(t *testing.T) {
	t.Run("Failed to login with incorrect password", func(t *testing.T) {
		resp, err := client.R().
			SetBody(auth.LoginPlainInput{
				Username: "poop",
				Password: "incorrect password",
			}).
			Post("/auth/login/plain")
		if err != nil || resp.StatusCode() != http.StatusUnauthorized {
			t.Logf("%+v\n", resp)
			t.Fatal("failed to fail incorrect password")
		}
	})

	t.Run("Login with the user", func(t *testing.T) {
		resp, err := client.R().
			SetBody(auth.LoginPlainInput{
				Username: "poop",
				Password: "poop",
			}).
			Post("/auth/login/plain")
		if resp.StatusCode() != http.StatusFound {
			t.Logf("%v\n", resp)
			t.Fatalf("failed to post /auth/login/plain, %v", err)
		}

		setCookie(client, resp)
	})
}

func Test_01_3_api_me(t *testing.T) {
	var result api.MeOutput
	resp, err := client.R().
		SetResult(&result).
		Get("/api/me")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%v\n", resp)
		t.Fatalf("failed to get /api/me: %v", err)
	}

	// Validate the response
	if result.AccessTokenClaim.Resource.Username != "poop" {
		t.Fatalf("unexpected username: got %s, want poop",
			result.AccessTokenClaim.Resource.Username)
	}

	if result.IsAdmin {
		t.Fatal("expected user to not be admin")
	}
}

func init() {
	viper.Set("admin-token", "poop")
}
