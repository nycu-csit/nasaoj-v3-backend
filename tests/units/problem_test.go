package units

import (
	"testing"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/parsers"
	"go.yaml.in/yaml/v3"
)

func TestProblem(t *testing.T) {
	input := "" +
		"stages:\n" +
		"  stage1:\n" +
		"    testcases:\n" +
		"      - name: test1\n" +
		"        file: poop.sh.tmpl"

	var problem parsers.Problem
	err := yaml.Unmarshal([]byte(input), &problem)
	if err != nil {
		t.Fatalf("expect no error, got %+v", err)
	}
	cooldown := problem.Cooldown
	if cooldown != 300 {
		t.Fatalf("expect Cooldown = 300, got %v", cooldown)
	}
	timeout := problem.Stages["stage1"].Testcases[0].Timeout
	if timeout != 30 {
		t.Fatalf("expect Timeout = 30, got %v", timeout)
	}

	tag := problem.Tag
	if tag != "default" {
		t.Fatalf("expect Tag = 'default', got %v", tag)
	}
}
