package units

import (
	"testing"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/utils"
	"gopkg.in/yaml.v3"
)

func TestCheckDeps(t *testing.T) {
	tests := []struct {
		Name          string
		ProblemString string
		ExpectedFail  bool
	}{
		{
			Name: "test with no cycle",
			ProblemString: `
---
name: Test deps with no cycle
problemId: 1
stages:
  stage1:
    before:
      - stage2
  stage2:
`,
			ExpectedFail: false,
		}, {
			Name: "test with cycle",
			ProblemString: `
---
name: Test deps with cycle
problemId: 2
stages:
  stage1:
    before:
      - stage2
  stage2:
    before:
      - stage1
`,
			ExpectedFail: true,
		},
	}

	tracing.InitTracer("nasaoj-test")
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			var problem models.Problem
			err := yaml.Unmarshal([]byte(test.ProblemString), &problem)
			if err != nil {
				t.Fatalf("unmarshal failed, %+v", err)
			}

			err = utils.CheckDeps(t.Context(), problem)
			t.Logf("Type: %T", err)
			if (err != nil) != test.ExpectedFail {
				t.Fatalf("expect no error, got %v", err)
			}
		})
	}
}
