package models

type Testcase struct {
	Name string            `json:"name" yaml:"name"`
	File string            `json:"file" yaml:"file"`
	Args map[string]string `json:"args" yaml:"args"`
}
