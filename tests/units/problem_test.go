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

func TestProblemWithContainerImage(t *testing.T) {
	input := `
stages:
  compile:
    testcases:
      - name: build
        image: "gcc:14"
        scripts:
          - gcc -o main main.c
    score: 50
  test:
    testcases:
      - name: run
        scripts:
          - ./main
    score: 50
`

	var problem parsers.Problem
	err := yaml.Unmarshal([]byte(input), &problem)
	if err != nil {
		t.Fatalf("expect no error, got %+v", err)
	}

	// Check compile stage has image set
	compileStage := problem.Stages["compile"]
	if len(compileStage.Testcases) != 1 {
		t.Fatalf("expect 1 testcase in compile stage, got %d", len(compileStage.Testcases))
	}

	buildTestcase := compileStage.Testcases[0]
	if buildTestcase.Image != "gcc:14" {
		t.Fatalf("expect Image = 'gcc:14', got '%s'", buildTestcase.Image)
	}

	// Check test stage has no image (empty string)
	testStage := problem.Stages["test"]
	if len(testStage.Testcases) != 1 {
		t.Fatalf("expect 1 testcase in test stage, got %d", len(testStage.Testcases))
	}

	runTestcase := testStage.Testcases[0]
	if runTestcase.Image != "" {
		t.Fatalf("expect Image = '', got '%s'", runTestcase.Image)
	}
}

func TestProblemContainerImageWithOtherFields(t *testing.T) {
	input := `
stages:
  main:
    testcases:
      - name: container-test
        image: "alpine:latest"
        timeout: 60
        resourceGroup: isolated
        scripts:
          - echo "running in container"
    score: 100
`

	var problem parsers.Problem
	err := yaml.Unmarshal([]byte(input), &problem)
	if err != nil {
		t.Fatalf("expect no error, got %+v", err)
	}

	testcase := problem.Stages["main"].Testcases[0]

	if testcase.Name != "container-test" {
		t.Fatalf("expect Name = 'container-test', got '%s'", testcase.Name)
	}

	if testcase.Image != "alpine:latest" {
		t.Fatalf("expect Image = 'alpine:latest', got '%s'", testcase.Image)
	}

	if testcase.Timeout != 60 {
		t.Fatalf("expect Timeout = 60, got %d", testcase.Timeout)
	}

	if testcase.ResourceGroup != "isolated" {
		t.Fatalf("expect ResourceGroup = 'isolated', got '%s'", testcase.ResourceGroup)
	}

	if len(testcase.Scripts) != 1 || testcase.Scripts[0] != "echo \"running in container\"" {
		t.Fatalf("expect Scripts = ['echo \"running in container\"'], got %v", testcase.Scripts)
	}
}
