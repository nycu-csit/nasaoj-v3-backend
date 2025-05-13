package models

type Testcase struct {
	Name   string            `json:"name" yaml:"name"`
	File   string            `json:"file" yaml:"file"`
	Args   map[string]string `json:"args" yaml:"args"`
	SoftTL int64             `json:"softTL" yaml:"softtl" default:"25"`
	HardTL int64             `json:"hardTL" yaml:"hardtl" default:"30"`
}
