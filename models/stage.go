package models

type StageName = string

type Stage struct {
	Testcases []Testcase        `json:"testcases" yaml:"testcases"`
	Before    []StageName       `json:"before" yaml:"before,omitempty"`
	Args      map[string]string `json:"args" yaml:"args,omitempty"`
}
