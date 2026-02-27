package tests

import (
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

// Shared state for tools tests
var (
	test09ToolSHA1         string
	test09TypedArgsSHA1    string
	test09ArgsOverrideSHA1 string
	test09UserId           int64
	test09SubmissionId     int64
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
		case 89:
			test09ArgsOverrideSHA1 = tool.SHA1
		}
	}

	t.Logf("Found %d tools, genpass SHA1: %s, typed-args SHA1: %s, args-override SHA1: %s",
		len(tools), test09ToolSHA1, test09TypedArgsSHA1, test09ArgsOverrideSHA1)
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

	test09SubmissionId = output.Id
	t.Logf("Submitted tool, submission id: %d", output.Id)
}

func Test_09_4a_Check_user_args(t *testing.T) {
	if test09SubmissionId == 0 || test09UserId == 0 {
		t.Skip("missing prerequisites from previous tests")
	}

	type ResultOutputJob struct {
		StageName  string `yaml:"stageName"`
		TestName   string `yaml:"testName"`
		Status     int64  `yaml:"status"`
		StatusName string `yaml:"statusName"`
	}

	type ResultOutput struct {
		Id   int64             `yaml:"id"`
		Jobs []ResultOutputJob `yaml:"jobs"`
	}

	var lastResult ResultOutput

	// Poll for job completion
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command(
			"judge", "admin", "get", "results", strconv.FormatInt(test09SubmissionId, 10),
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to get results: %v\n", err)
		}

		if err := yaml.Unmarshal([]byte(stdout.String()), &lastResult); err != nil {
			t.Fatalf("failed to unmarshal yaml: %v", err)
		}

		// Check if all jobs are done (status >= 262 means completed, 262=Success)
		allDone := len(lastResult.Jobs) > 0
		for _, job := range lastResult.Jobs {
			if job.Status < 262 {
				allDone = false
				break
			}
		}

		if allDone {
			break
		}

		if i == maxRetries-1 {
			t.Fatalf("jobs did not complete within timeout, last status: %+v", lastResult.Jobs)
		}
		time.Sleep(time.Second)
	}

	// Log job status and output
	for _, job := range lastResult.Jobs {
		t.Logf("Job %s/%s status: %s (%d)", job.StageName, job.TestName, job.StatusName, job.Status)

		// Always get stdout/stderr for debugging
		stdoutCmd := exec.Command(
			"judge", "admin", "get", "stdout",
			strconv.FormatInt(test09SubmissionId, 10), job.StageName, job.TestName,
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")
		var stdoutOut strings.Builder
		stdoutCmd.Stdout = &stdoutOut
		stdoutCmd.Run()
		t.Logf("Job stdout:\n%s", stdoutOut.String())

		stderrCmd := exec.Command(
			"judge", "admin", "get", "stderr",
			strconv.FormatInt(test09SubmissionId, 10), job.StageName, job.TestName,
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")
		var stderrOut strings.Builder
		stderrCmd.Stdout = &stderrOut
		stderrCmd.Run()
		t.Logf("Job stderr:\n%s", stderrOut.String())
	}

	// Get user resource and check the args set by the testcase script
	cmd := exec.Command(
		"judge", "admin", "get", "user", strconv.FormatInt(test09UserId, 10),
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
		t.Fatalf("failed to get user: %v\n", err)
	}

	type UserResource struct {
		Id   int64             `yaml:"id"`
		Args map[string]string `yaml:"args"`
	}

	var user UserResource
	if err := yaml.Unmarshal([]byte(stdout.String()), &user); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	// The testcase script (test-userargs.sh.tmpl) sets:
	// 1. {"pass": "<random>"} via PUT /api/user
	// 2. {"poop": "poop"} via PUT /api/user
	// Both should exist (different args are preserved)

	if user.Args["poop"] != "poop" {
		t.Fatalf("expected user.Args['poop'] = %q, got %q (all args: %v)", "poop", user.Args["poop"], user.Args)
	}

	// Verify 'pass' arg also exists (set by first PUT call)
	if user.Args["pass"] == "" {
		t.Fatalf("expected user.Args['pass'] to exist, got empty (all args: %v)", user.Args)
	}

	t.Logf("User args verified: poop=%s, pass=%s (different args preserved)", user.Args["poop"], user.Args["pass"])
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

func Test_09_9_Enable_args_override_tool(t *testing.T) {
	if test09ArgsOverrideSHA1 == "" {
		t.Skip("no args-override tool SHA1 available from previous test")
	}

	cmd := exec.Command(
		"judge", "admin", "enable", "tool", test09ArgsOverrideSHA1,
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
		t.Fatalf("failed to enable args-override tool: %v\n", err)
	}

	t.Logf("Enable args-override tool output: %s", stdout.String())
}

func Test_09_9a_Submit_args_override_tool(t *testing.T) {
	if test09ArgsOverrideSHA1 == "" || test09UserId == 0 {
		t.Skip("missing prerequisites from previous tests")
	}

	cmd := exec.Command(
		"judge", "admin", "submit", "tool", test09ArgsOverrideSHA1,
		"--owner-id", strconv.FormatInt(test09UserId, 10),
		"--arg", "dummy=test",
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
		t.Fatalf("failed to submit args-override tool: %v\n", err)
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

	// Store submission ID for the next test
	test09SubmissionId = output.Id
	t.Logf("Submitted args-override tool, submission id: %d", output.Id)
}

func Test_09_9b_Check_args_override(t *testing.T) {
	if test09SubmissionId == 0 || test09UserId == 0 {
		t.Skip("missing prerequisites from previous tests")
	}

	type ResultOutputJob struct {
		StageName  string `yaml:"stageName"`
		TestName   string `yaml:"testName"`
		Status     int64  `yaml:"status"`
		StatusName string `yaml:"statusName"`
	}

	type ResultOutput struct {
		Id   int64             `yaml:"id"`
		Jobs []ResultOutputJob `yaml:"jobs"`
	}

	var lastResult ResultOutput

	// Poll for job completion
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		cmd := exec.Command(
			"judge", "admin", "get", "results", strconv.FormatInt(test09SubmissionId, 10),
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to get results: %v\n", err)
		}

		if err := yaml.Unmarshal([]byte(stdout.String()), &lastResult); err != nil {
			t.Fatalf("failed to unmarshal yaml: %v", err)
		}

		// Check if all jobs are done (status >= 262 means completed)
		allDone := len(lastResult.Jobs) > 0
		for _, job := range lastResult.Jobs {
			if job.Status < 262 {
				allDone = false
				break
			}
		}

		if allDone {
			break
		}

		if i == maxRetries-1 {
			t.Fatalf("jobs did not complete within timeout, last status: %+v", lastResult.Jobs)
		}
		time.Sleep(time.Second)
	}

	// Log job status
	for _, job := range lastResult.Jobs {
		t.Logf("Job %s/%s status: %s (%d)", job.StageName, job.TestName, job.StatusName, job.Status)

		// If job failed, get output for debugging
		if job.Status >= 460 {
			stdoutCmd := exec.Command(
				"judge", "admin", "get", "stdout",
				strconv.FormatInt(test09SubmissionId, 10), job.StageName, job.TestName,
				"--access-token", viper.GetString("admin-token"),
				"--endpoint", "http://localhost:8080")
			var stdoutOut strings.Builder
			stdoutCmd.Stdout = &stdoutOut
			stdoutCmd.Run()
			t.Logf("Job stdout:\n%s", stdoutOut.String())

			stderrCmd := exec.Command(
				"judge", "admin", "get", "stderr",
				strconv.FormatInt(test09SubmissionId, 10), job.StageName, job.TestName,
				"--access-token", viper.GetString("admin-token"),
				"--endpoint", "http://localhost:8080")
			var stderrOut strings.Builder
			stderrCmd.Stdout = &stderrOut
			stderrCmd.Run()
			t.Logf("Job stderr:\n%s", stderrOut.String())
		}
	}

	// Get user resource and check the args
	cmd := exec.Command(
		"judge", "admin", "get", "user", strconv.FormatInt(test09UserId, 10),
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
		t.Fatalf("failed to get user: %v\n", err)
	}

	type UserResource struct {
		Id   int64             `yaml:"id"`
		Args map[string]string `yaml:"args"`
	}

	var user UserResource
	if err := yaml.Unmarshal([]byte(stdout.String()), &user); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	// Test 1: Override - 'poop' should be overridden from "poop" to "overridden"
	if user.Args["poop"] != "overridden" {
		t.Errorf("expected user.Args['poop'] = %q (overridden), got %q", "overridden", user.Args["poop"])
	}

	// Test 2: Preserve - 'pass' should still exist from the previous tool run
	if user.Args["pass"] == "" {
		t.Errorf("expected user.Args['pass'] to be preserved, got empty")
	}

	// Test 3: New arg - 'newarg' should exist from this tool run
	if user.Args["newarg"] != "hello" {
		t.Errorf("expected user.Args['newarg'] = %q, got %q", "hello", user.Args["newarg"])
	}

	t.Logf("Args override test passed: poop=%s (overridden), pass=%s (preserved), newarg=%s (new)",
		user.Args["poop"], user.Args["pass"], user.Args["newarg"])
}
