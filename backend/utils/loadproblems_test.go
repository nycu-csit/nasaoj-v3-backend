package utils

import (
	"testing"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gopkg.in/yaml.v3"
)

func TestProblem(t *testing.T) {
	input := "" +
		"stages:\n" +
		"  stage1:\n" +
		"    testcases:\n" +
		"      - name: test1\n" +
		"        file: poop.sh.tmpl"

	var problem models.Problem
	err := yaml.Unmarshal([]byte(input), &problem)
	if err != nil {
		t.Fatalf("expect no error, got %+v", err)
	}
	softTL := problem.Stages["stage1"].Testcases[0].SoftTL
	if softTL != 25 {
		t.Fatalf("expect SoftTL = 25, got %v", softTL)
	}
}
