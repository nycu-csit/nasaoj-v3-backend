package models

// The minimal unit to submit
type Problem struct {
	Name        string            `json:"name" yaml:"name"`
	ProblemId   int64             `json:"problemId" yaml:"problemId"`
	Stages      map[string]Stage  `json:"stages" yaml:"stages"`
	Args        map[string]string `json:"args" yaml:"args"`
	Tag         string            `json:"tag" yaml:"tag"`
	PreScripts  []string          `json:"preScripts" yaml:"preScripts"`
	PostScripts []string          `json:"postScripts" yaml:"postScripts"`
}
