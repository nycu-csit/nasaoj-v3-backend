package models

import (
	"gopkg.in/yaml.v3"
)

type TestcaseNode struct {
	SoftTL yaml.Node `yaml:"softtl"`
	HardTL yaml.Node `yaml:"hardtl"`
}

type Testcase struct {
	Name   string            `json:"name" yaml:"name"`
	File   string            `json:"file" yaml:"file"`
	Args   map[string]string `json:"args" yaml:"args"`
	SoftTL int64             `json:"softTL" yaml:"softtl"`
	HardTL int64             `json:"hardTL" yaml:"hardtl"`
}

type testcase struct {
	Testcase `yaml:",inline"`
}

func (self *Testcase) UnmarshalYAML(v *yaml.Node) error {
	var node TestcaseNode
	err := v.Decode(&node)
	if err != nil {
		return err
	}

	var im testcase
	err = v.Decode(im)
	if err != nil {
		return err
	}

	if node.SoftTL.IsZero() {
		self.SoftTL = 25
	}
	if node.HardTL.IsZero() {
		self.HardTL = 30
	}
	self = &im.Testcase

	return nil
}
