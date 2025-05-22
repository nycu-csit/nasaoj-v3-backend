package models

type StageName = string

// Stage can have dependencies
type Stage struct {
	Testcases  []Testcase        `json:"testcases" yaml:"testcases"`
	Before     []StageName       `json:"before" yaml:"before,omitempty"`
	Args       map[string]string `json:"args" yaml:"args,omitempty"`
	PreScript  Content           `json:"preScript" yaml:"preScript"`
	PostScript Content           `json:"postScript" yaml:"postScript"`
	Score      int64             `json:"score" yaml:"score"`
}
