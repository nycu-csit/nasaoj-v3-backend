package tests

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

func Test_07_1_Get_workers(t *testing.T) {
	cmd := exec.Command(
		"judge", "admin", "get", "workers",
		"--access-token", viper.GetString("admin-token"),
		"--endpoint", "http://localhost:8080")

	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to get workers: %v\n", err)
	}

	type Worker struct {
		Name       string   `yaml:"name"`
		Ip         string   `yaml:"ip"`
		Tags       []string `yaml:"tags"`
		LastAccess string   `yaml:"lastAccess"`
	}

	var workers []Worker
	if err := yaml.Unmarshal([]byte(stdout.String()), &workers); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	// We should have at least one worker running for e2e tests
	if len(workers) == 0 {
		t.Fatal("expected at least one worker")
	}

	t.Logf("Found %d workers", len(workers))
	for _, w := range workers {
		t.Logf("  - %s (%s) tags=%v lastAccess=%s", w.Name, w.Ip, w.Tags, w.LastAccess)
	}
}
