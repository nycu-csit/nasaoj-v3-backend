package models

type StageName = string

// Stage can have dependencies
type Stage struct {
	Testcases []Testcase        `json:"testcases" yaml:"testcases"`
	Before    []StageName       `json:"before" yaml:"before,omitempty"`
	Args      map[string]string `json:"args" yaml:"args,omitempty"`

	Score int64 `json:"score" yaml:"score"`
}
