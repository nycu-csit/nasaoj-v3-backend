package tests

import (
	"os/exec"
	"sort"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

var (
	problemSHA1 string
)

func Test_02_1_Import_problem(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "import", "problems",
		"--problems-dir", "resource/problems",
		"--testcases-dir", "resource/testcases",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Logf("%s\n", out.String())
		t.Fatalf("finish with %v\n", err)
	}
}

func Test_02_2_Enable_problem(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "get", "problems",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("finish with %v\n", err)
	}

	type Problem struct {
		SHA1       string `yaml:"sha1"`
		ImportedAt string `yaml:"importedAt"`
	}

	var problems []Problem
	if err := yaml.Unmarshal([]byte(stdout.String()), &problems); err != nil {
		t.Fatalf("failed to unmarshal to yaml")
	}

	if len(problems) <= 0 {
		t.Fatalf("problems not imported")
	}

	sort.Slice(problems, func(i, j int) bool {
		return problems[i].ImportedAt > problems[j].ImportedAt
	})
	problemSHA1 = problems[0].SHA1

	cmd = exec.Command(
		"judge", "admin", "enable", "problem", problemSHA1,
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	if err := cmd.Run(); err != nil {
		t.Fatalf("finish with %v\n", err)
	}
}
