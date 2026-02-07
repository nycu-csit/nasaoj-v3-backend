package tests

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

// Shared state for token tests
var (
	test10AdminToken       string
	test10AdminTokenSHA256 string
	test10WorkerToken      string
)

// hashToken computes SHA256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func Test_10_1_Gen_tokens(t *testing.T) {
	t.Run("gen admin token", func(t *testing.T) {
		cmd := exec.Command(
			"judge", "admin", "gen", "token", "admin",
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to gen admin token: %v\n", err)
		}

		output := stdout.String()
		// Extract token from "Token: <token>"
		re := regexp.MustCompile(`Token: (\S+)`)
		matches := re.FindStringSubmatch(output)
		if len(matches) < 2 {
			t.Fatalf("failed to parse token from output: %s", output)
		}
		test10AdminToken = matches[1]
		test10AdminTokenSHA256 = hashToken(test10AdminToken)
		t.Logf("Generated admin token: %s (SHA256: %s)", test10AdminToken, test10AdminTokenSHA256)
	})

	t.Run("gen worker token", func(t *testing.T) {
		cmd := exec.Command(
			"judge", "admin", "gen", "token", "worker",
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to gen worker token: %v\n", err)
		}

		output := stdout.String()
		re := regexp.MustCompile(`Token: (\S+)`)
		matches := re.FindStringSubmatch(output)
		if len(matches) < 2 {
			t.Fatalf("failed to parse token from output: %s", output)
		}
		test10WorkerToken = matches[1]
		t.Logf("Generated worker token: %s", test10WorkerToken)
	})
}

func Test_10_2_Get_tokens(t *testing.T) {
	if test10AdminTokenSHA256 == "" {
		t.Skip("no admin token SHA256 available from previous test")
	}

	cmd := exec.Command(
		"judge", "admin", "get", "tokens",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get tokens: %v\n", err)
	}

	type Token struct {
		SHA256    string `yaml:"sha256"`
		IsEnabled bool   `yaml:"isEnabled"`
		CreatedAt string `yaml:"createdAt"`
	}

	var tokens []Token
	if err := yaml.Unmarshal([]byte(stdout.String()), &tokens); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	// Should have at least the tokens we generated
	if len(tokens) < 2 {
		t.Fatalf("expected at least 2 tokens, got %d", len(tokens))
	}

	// Verify our generated token is in the list
	found := false
	for _, token := range tokens {
		if token.SHA256 == test10AdminTokenSHA256 {
			found = true
			t.Logf("Found generated admin token (SHA256: %s, CreatedAt: %s)", token.SHA256, token.CreatedAt)
			break
		}
	}
	if !found {
		t.Fatalf("generated admin token not found in tokens list")
	}

	t.Logf("Found %d tokens", len(tokens))
}

func Test_10_3_Disable_enable_token(t *testing.T) {
	if test10AdminToken == "" {
		t.Skip("no admin token available from previous test")
	}

	type Token struct {
		SHA256    string `yaml:"sha256"`
		IsEnabled bool   `yaml:"isEnabled"`
	}

	t.Run("verify token works before disable", func(t *testing.T) {
		// Use the generated token to make an API call
		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+test10AdminToken).
			Get("/api/admin/tokens")
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode() != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode())
		}
		t.Log("Token works before disable")
	})

	t.Run("disable token", func(t *testing.T) {
		cmd := exec.Command(
			"judge", "admin", "disable", "token", test10AdminToken,
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var (
			stdout strings.Builder
			stderr strings.Builder
		)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Logf("stderr: %s\n", stderr.String())
			t.Fatalf("failed to disable token: %v\n", err)
		}

		t.Logf("Disable token output: %s", stdout.String())
	})

	t.Run("verify token disabled in list", func(t *testing.T) {
		cmd := exec.Command(
			"judge", "admin", "get", "tokens",
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to get tokens: %v\n", err)
		}

		var tokens []Token
		if err := yaml.Unmarshal([]byte(stdout.String()), &tokens); err != nil {
			t.Fatalf("failed to unmarshal yaml: %v", err)
		}

		for _, token := range tokens {
			if token.SHA256 == test10AdminTokenSHA256 {
				if token.IsEnabled {
					t.Fatal("expected token to be disabled")
				}
				return
			}
		}
		t.Fatal("token not found")
	})

	t.Run("verify disabled token rejected", func(t *testing.T) {
		// Use the disabled token to make an API call - should fail
		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+test10AdminToken).
			Get("/api/admin/tokens")
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode() == http.StatusOK {
			t.Fatal("expected disabled token to be rejected, but request succeeded")
		}
		t.Logf("Disabled token correctly rejected with status %d", resp.StatusCode())
	})

	t.Run("enable token", func(t *testing.T) {
		cmd := exec.Command(
			"judge", "admin", "enable", "token", test10AdminToken,
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var (
			stdout strings.Builder
			stderr strings.Builder
		)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			t.Logf("stderr: %s\n", stderr.String())
			t.Fatalf("failed to enable token: %v\n", err)
		}

		t.Logf("Enable token output: %s", stdout.String())
	})

	t.Run("verify token enabled in list", func(t *testing.T) {
		cmd := exec.Command(
			"judge", "admin", "get", "tokens",
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to get tokens: %v\n", err)
		}

		var tokens []Token
		if err := yaml.Unmarshal([]byte(stdout.String()), &tokens); err != nil {
			t.Fatalf("failed to unmarshal yaml: %v", err)
		}

		for _, token := range tokens {
			if token.SHA256 == test10AdminTokenSHA256 {
				if !token.IsEnabled {
					t.Fatal("expected token to be enabled")
				}
				return
			}
		}
		t.Fatal("token not found")
	})

	t.Run("verify enabled token works", func(t *testing.T) {
		// Use the re-enabled token to make an API call - should work
		resp, err := client.R().
			SetHeader("Authorization", "Bearer "+test10AdminToken).
			Get("/api/admin/tokens")
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode() != http.StatusOK {
			t.Fatalf("expected status 200, got %d", resp.StatusCode())
		}
		t.Log("Re-enabled token works")
	})
}
