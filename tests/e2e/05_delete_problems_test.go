package tests

import (
	"net/http"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/backend/handlers/api"
)

func Test_05_Delete_problems(t *testing.T) {
	for _, pres := range results {
		t.Run(pres.SHA1, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command(
				"judge", "admin", "delete", "problem", pres.SHA1,
				"--access-token", viper.GetString("admin-token"),
				"--endpoint", "http://localhost:8080",
			)
			var out strings.Builder
			cmd.Stdout = &out
			cmd.Stderr = &out
			if err := cmd.Run(); err != nil {
				t.Logf("output: %s\n", out.String())
				t.Fatalf("failed to delete problem %d (SHA1: %s): %v", pres.Id, pres.SHA1, err)
			}
		})
	}
}

func Test_06_Check_problems(t *testing.T) {
	var res []api.ProblemOutput
	resp, err := client.R().
		SetResult(&res).
		Get("/api/problems")
	if err != nil || resp.StatusCode() != http.StatusOK {
		t.Fatalf("failed to get /api/problems: %v", resp)
	}

	for _, pres := range results {
		for _, problem := range res {
			if problem.SHA1 == pres.SHA1 {
				t.Fatalf("problem %d (SHA1: %s) was not deleted", pres.Id, pres.SHA1)
			}
		}
	}
}
