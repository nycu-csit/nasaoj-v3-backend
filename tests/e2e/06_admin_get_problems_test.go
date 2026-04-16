package tests

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

func Test_06_1_Import_problems_for_dup_test(t *testing.T) {
	ImportProblems(t, ProblemsDir(6), TestcasesDir(6))
}

func Test_06_2_Get_problems_dup(t *testing.T) {
	type Problem struct {
		ProblemId int64 `yaml:"problemId"`
	}

	// Test without --all flag (should show max 3 per problemId)
	cmd := exec.Command(
		"judge", "admin", "get", "problems",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get problems: %v\n", err)
	}

	var problems []Problem
	if err := yaml.Unmarshal([]byte(stdout.String()), &problems); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	countWithoutAll := 0
	for _, p := range problems {
		if p.ProblemId == 601 {
			countWithoutAll++
		}
	}

	if countWithoutAll != 3 {
		t.Fatalf("expected 3 problems with problemId=601 without --all, got %d",
			countWithoutAll)
	}

	// Test with --all flag (should show all 4)
	cmd = exec.Command(
		"judge", "admin", "get", "problems", "--all",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	stdout.Reset()
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get problems with --all: %v\n", err)
	}

	if err := yaml.Unmarshal([]byte(stdout.String()), &problems); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	countWithAll := 0
	for _, p := range problems {
		if p.ProblemId == 601 {
			countWithAll++
		}
	}

	if countWithAll != 4 {
		t.Fatalf("expected 4 problems with problemId=601 with --all, got %d",
			countWithAll)
	}
}
