package models

type Problem struct {
	Name   string  `json:"name" yaml:"name"`
	Stages []Stage `json:"stages" yaml:"stages"`
}

type ProblemHydrated struct {
	Name   string          `json:"name"`
	Stages []StageHydrated `json:"stages"`
}
