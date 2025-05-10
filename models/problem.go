package models

// The minimal unit to submit
type Problem struct {
	Name      string            `json:"name" yaml:"name"`
	ProblemId int               `json:"problemId" yaml:"problemId"`
	Stages    map[string]Stage  `json:"stages" yaml:"stages"`
	Args      map[string]string `json:"args" yaml:"args"`
}
