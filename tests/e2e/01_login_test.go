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

func Test_01_0_Create_CSSSO_Users(t *testing.T) {
	csssoUsers := []api.PutUsersCSSSOInputUser{
		{CsId: "dummy_user_1"},
		{CsId: "dummy_user_2"},
		{CsId: "dummy_user_3"},
		{CsId: "dummy_user_4"},
		{CsId: "dummy_user_5"},
	}

	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+viper.GetString("admin-token")).
		SetBody(csssoUsers).
		Put("/api/admin/users/cssso")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Logf("%+v\n", resp)
		t.Fatalf("failed to create CSSSO users: %v", err)
	}
}

func Test_01_1_Create_User(t *testing.T) {
	t.Logf("access with admin token: %s\n", viper.GetString("admin-token"))

	stdin := strings.NewReader(`---
- username: 'poop'
  password: 'poop'
- username: 'admin'
  password: 'poop'
  isAdmin: true
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
		if resp.StatusCode() != http.StatusOK {
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

func Test_01_4_Update_Password(t *testing.T) {
	newPassword := "newpassword123"

	t.Run("Update user password via admin API", func(t *testing.T) {
		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+viper.GetString("admin-token")).
			SetBody([]api.PutUsersPlainInputUser{
				{
					Username: "poop",
					Password: newPassword,
				},
			}).
			Put("/api/admin/users/plain")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Logf("%+v\n", resp)
			t.Fatalf("failed to update user password: %v", err)
		}
	})

	t.Run("Login with old password should fail", func(t *testing.T) {
		resp, err := client.R().
			SetBody(auth.LoginPlainInput{
				Username: "poop",
				Password: "poop",
			}).
			Post("/auth/login/plain")
		if err != nil || resp.StatusCode() != http.StatusUnauthorized {
			t.Logf("%+v\n", resp)
			t.Fatal("expected login with old password to fail")
		}
	})

	t.Run("Login with new password should succeed", func(t *testing.T) {
		resp, err := client.R().
			SetBody(auth.LoginPlainInput{
				Username: "poop",
				Password: newPassword,
			}).
			Post("/auth/login/plain")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Logf("%+v\n", resp)
			t.Fatalf("failed to login with new password: %v", err)
		}

		setCookie(client, resp)
	})

	t.Run("Revert password back to original", func(t *testing.T) {
		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+viper.GetString("admin-token")).
			SetBody([]api.PutUsersPlainInputUser{
				{
					Username: "poop",
					Password: "poop",
				},
			}).
			Put("/api/admin/users/plain")
		if err != nil || resp.StatusCode() != http.StatusOK {
			t.Logf("%+v\n", resp)
			t.Fatalf("failed to revert user password: %v", err)
		}
	})
}

func init() {
	viper.Set("admin-token", "poop")
}
