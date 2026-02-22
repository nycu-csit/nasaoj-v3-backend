package tests

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

func Test_11_1_Get_user(t *testing.T) {
	// Get the user ID for the "poop" user created in test 01
	cmd := exec.Command(
		"judge", "admin", "get", "users",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get users: %v\n", err)
	}

	type User struct {
		Id       int64  `yaml:"id"`
		Username string `yaml:"username"`
	}

	var users []User
	if err := yaml.Unmarshal([]byte(stdout.String()), &users); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	var userId int64
	for _, u := range users {
		if u.Username == "poop" {
			userId = u.Id
			break
		}
	}

	if userId == 0 {
		t.Fatal("user 'poop' not found")
	}

	// Get the specific user
	cmd = exec.Command(
		"judge", "admin", "get", "user", strconv.FormatInt(userId, 10),
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get user: %v\n", err)
	}

	type UserResource struct {
		Id       int64  `yaml:"id"`
		Username string `yaml:"username"`
	}

	var user UserResource
	if err := yaml.Unmarshal([]byte(stdout.String()), &user); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if user.Username != "poop" {
		t.Fatalf("expected username 'poop', got '%s'", user.Username)
	}

	t.Logf("Got user: id=%d username=%s", user.Id, user.Username)
}

func Test_11_2_Delete_user(t *testing.T) {
	// Create a temporary user to delete
	stdin := strings.NewReader(`---
- username: 'tempuser'
  password: 'temppass'
`)

	cmd := exec.Command(
		"judge", "admin", "import", "users", "plain",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080",
	)
	cmd.Stdin = stdin
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Logf("output: %s\n", out.String())
		t.Fatalf("failed to create temp user: %v", err)
	}

	// Get the user ID
	cmd = exec.Command(
		"judge", "admin", "get", "users",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get users: %v\n", err)
	}

	type User struct {
		Id       int64  `yaml:"id"`
		Username string `yaml:"username"`
	}

	var users []User
	if err := yaml.Unmarshal([]byte(stdout.String()), &users); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	var tempUserId int64
	for _, u := range users {
		if u.Username == "tempuser" {
			tempUserId = u.Id
			break
		}
	}

	if tempUserId == 0 {
		t.Fatal("temp user not found")
	}

	// Delete the user
	cmd = exec.Command(
		"judge", "admin", "delete", "user", strconv.FormatInt(tempUserId, 10),
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	stdout.Reset()
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Logf("stderr: %s\n", stderr.String())
		t.Fatalf("failed to delete user: %v\n", err)
	}

	t.Logf("Delete user output: %s", stdout.String())

	// Verify user is deleted
	cmd = exec.Command(
		"judge", "admin", "get", "users",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get users: %v\n", err)
	}

	if err := yaml.Unmarshal([]byte(stdout.String()), &users); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	for _, u := range users {
		if u.Username == "tempuser" {
			t.Fatal("temp user should have been deleted")
		}
	}

	t.Log("User successfully deleted")
}
