package models

type Content struct {
	Files   []string `json:"files" yaml:"files"`
	Scripts []string `json:"scripts" yaml:"scripts"`
}
