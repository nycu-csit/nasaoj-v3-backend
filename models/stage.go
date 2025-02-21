package models

type StageName = string

type Stage struct {
	Name      StageName   `json:"name" yaml:"name"`
	Testcases []string    `json:"testcases" yaml:"testcases"`
	Before    []StageName `json:"before" yaml:"before,omitempty"`
}

type StageHydrated struct {
	Name      StageName   `json:"name"`
	Testcases []Job       `json:"testcases"`
	Before    []StageName `json:"before"`
}
