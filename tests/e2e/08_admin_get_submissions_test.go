package tests

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

func Test_08_1_Get_submissions_with_filters(t *testing.T) {
	type Submission struct {
		Id      int64  `yaml:"id"`
		RefSHA1 string `yaml:"refSHA1"`
		RefId   int64  `yaml:"refId"`
		Type    int    `yaml:"type"`
		OwnerId string `yaml:"ownerId"`
	}

	// Test get submissions without filters
	t.Run("no filters", func(t *testing.T) {
		cmd := exec.Command(
			"judge", "admin", "get", "submissions",
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to get submissions: %v\n", err)
		}

		var submissions []Submission
		// Output includes "page: x/y" at the end, so we need to filter it out
		lines := strings.Split(stdout.String(), "\n")
		var yamlLines []string
		for _, line := range lines {
			if !strings.HasPrefix(line, "page:") {
				yamlLines = append(yamlLines, line)
			}
		}

		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &submissions); err != nil {
			t.Fatalf("failed to unmarshal yaml: %v", err)
		}

		if len(submissions) == 0 {
			t.Fatal("expected at least one submission")
		}
		t.Logf("Found %d submissions without filters", len(submissions))
	})

	// Test get submissions with ref-id filter
	t.Run("with ref-id filter", func(t *testing.T) {
		cmd := exec.Command(
			"judge", "admin", "get", "submissions",
			"--ref-id", "1",
			"--type", "0",
			"--access-token", viper.GetString("admin-token"),
			"--endpoint", "http://localhost:8080")

		var stdout strings.Builder
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to get submissions with ref-id: %v\n", err)
		}

		var submissions []Submission
		lines := strings.Split(stdout.String(), "\n")
		var yamlLines []string
		for _, line := range lines {
			if !strings.HasPrefix(line, "page:") {
				yamlLines = append(yamlLines, line)
			}
		}

		if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &submissions); err != nil {
			t.Fatalf("failed to unmarshal yaml: %v", err)
		}

		// All submissions should have refId = 1
		for _, sub := range submissions {
			if sub.RefId != 1 {
				t.Fatalf("expected refId=1, got %d", sub.RefId)
			}
		}
		t.Logf("Found %d submissions with ref-id=1", len(submissions))
	})
}
