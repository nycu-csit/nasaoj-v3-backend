package tests

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

// Shared state for tools tests
var (
	test09ToolSHA1      string
	test09TypedArgsSHA1 string
	test09UserId        int64
)

func Test_09_1_Import_tools(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "tools",
		"--tools-dir", "resources/02/tools",
		"--testcases-dir", "resources/02/testcases",
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
		t.Fatalf("failed to import tools: %v\n", err)
	}

	t.Logf("Import tools output: %s", stdout.String())
}

func Test_09_2_Get_tools(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "get", "tools",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get tools: %v\n", err)
	}

	type Tool struct {
		ToolId      int64  `yaml:"toolId"`
		SHA1        string `yaml:"sha1"`
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
		IsEnabled   bool   `yaml:"isEnabled"`
	}

	var tools []Tool
	if err := yaml.Unmarshal([]byte(stdout.String()), &tools); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if len(tools) == 0 {
		t.Fatal("expected at least one tool")
	}

	// Save tool SHA1s for later tests
	for _, tool := range tools {
		switch tool.ToolId {
		case 87:
			test09ToolSHA1 = tool.SHA1
		case 88:
			test09TypedArgsSHA1 = tool.SHA1
		}
	}

	t.Logf("Found %d tools, genpass SHA1: %s, typed-args SHA1: %s",
		len(tools), test09ToolSHA1, test09TypedArgsSHA1)
}

func Test_09_3_Enable_tool(t *testing.T) {
	if test09ToolSHA1 == "" {
		t.Skip("no tool SHA1 available from previous test")
	}

	cmd := exec.Command(
		"judge", "admin", "enable", "tool", test09ToolSHA1,
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
		t.Fatalf("failed to enable tool: %v\n", err)
	}

	t.Logf("Enable tool output: %s", stdout.String())
}

func Test_09_4_Submit_tool(t *testing.T) {
	if test09ToolSHA1 == "" {
		t.Skip("no tool SHA1 available from previous test")
	}

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

	for _, u := range users {
		if u.Username == "poop" {
			test09UserId = u.Id
			break
		}
	}

	if test09UserId == 0 {
		t.Fatal("user 'poop' not found")
	}

	// Submit tool as user
	cmd = exec.Command(
		"judge", "admin", "submit", "tool", test09ToolSHA1,
		"--owner-id", strconv.FormatInt(test09UserId, 10),
		"--arg", "test=poop",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	stdout.Reset()
	var stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Logf("stderr: %s\n", stderr.String())
		t.Fatalf("failed to submit tool: %v\n", err)
	}

	type SubmitOutput struct {
		Id      int64  `yaml:"id"`
		RefSHA1 string `yaml:"refSHA1"`
	}

	var output SubmitOutput
	if err := yaml.Unmarshal([]byte(stdout.String()), &output); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if output.Id == 0 {
		t.Fatal("submission id is 0")
	}

	t.Logf("Submitted tool, submission id: %d", output.Id)
}

func Test_09_5_Enable_typed_args_tool(t *testing.T) {
	if test09TypedArgsSHA1 == "" {
		t.Skip("no typed-args tool SHA1 available from previous test")
	}

	cmd := exec.Command(
		"judge", "admin", "enable", "tool", test09TypedArgsSHA1,
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
		t.Fatalf("failed to enable typed-args tool: %v\n", err)
	}

	t.Logf("Enable typed-args tool output: %s", stdout.String())
}

func Test_09_6_Submit_with_invalid_numeric(t *testing.T) {
	if test09TypedArgsSHA1 == "" || test09UserId == 0 {
		t.Skip("missing prerequisites from previous tests")
	}

	output, err := exec.Command(
		"judge", "admin", "submit", "tool", test09TypedArgsSHA1,
		"--owner-id", strconv.FormatInt(test09UserId, 10),
		"--arg", "count=not-a-number",
		"--arg", "enabled=true",
		"--arg", "message=hello",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080").CombinedOutput()

	if err == nil {
		t.Fatal("expected submission to fail with invalid numeric input")
	}

	if !strings.Contains(string(output), "numeric") {
		t.Fatalf("expected error message to mention 'numeric', got: %s", output)
	}
}

func Test_09_7_Submit_with_invalid_boolean(t *testing.T) {
	if test09TypedArgsSHA1 == "" || test09UserId == 0 {
		t.Skip("missing prerequisites from previous tests")
	}

	output, err := exec.Command(
		"judge", "admin", "submit", "tool", test09TypedArgsSHA1,
		"--owner-id", strconv.FormatInt(test09UserId, 10),
		"--arg", "count=42",
		"--arg", "enabled=not-a-boolean",
		"--arg", "message=hello",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080").CombinedOutput()

	if err == nil {
		t.Fatal("expected submission to fail with invalid boolean input")
	}

	if !strings.Contains(string(output), "boolean") {
		t.Fatalf("expected error message to mention 'boolean', got: %s", output)
	}
}

func Test_09_8_Submit_with_valid_typed_inputs(t *testing.T) {
	if test09TypedArgsSHA1 == "" || test09UserId == 0 {
		t.Skip("missing prerequisites from previous tests")
	}

	cmd := exec.Command(
		"judge", "admin", "submit", "tool", test09TypedArgsSHA1,
		"--owner-id", strconv.FormatInt(test09UserId, 10),
		"--arg", "count=42",
		"--arg", "enabled=true",
		"--arg", "message=hello",
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
		t.Fatalf("failed to submit tool with valid inputs: %v\n", err)
	}

	type SubmitOutput struct {
		Id      int64  `yaml:"id"`
		RefSHA1 string `yaml:"refSHA1"`
	}

	var output SubmitOutput
	if err := yaml.Unmarshal([]byte(stdout.String()), &output); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	if output.Id == 0 {
		t.Fatal("submission id is 0")
	}

	t.Logf("Successfully submitted tool with valid inputs, submission id: %d", output.Id)
}
