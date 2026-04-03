package tests

import (
	"testing"
)

var test02SHA1s []string

func Test_02_1_Import_and_enable_problems(t *testing.T) {
	result := ImportAndEnableProblems(t, ProblemsDir(2), TestcasesDir(2))
	test02SHA1s = result.SHA1s
	t.Logf("Imported and enabled %d problems", len(test02SHA1s))
}
