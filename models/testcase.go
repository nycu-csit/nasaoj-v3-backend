package models

import (
	"text/template"

	"gopkg.in/yaml.v3"
)

type TestcaseRepository = map[string]*template.Template

type testcase struct {
	Content `json:",inline" yaml:",inline"`

	Name   string            `json:"name" yaml:"name"`
	Args   map[string]string `json:"args" yaml:"args"`
	SoftTL int64             `json:"softTL" yaml:"softtl"`
	HardTL int64             `json:"hardTL" yaml:"hardtl"`
}

type Testcase struct {
	testcase `yaml:",inline"`
}

type testcaseNode struct {
	SoftTL yaml.Node `yaml:"softtl"`
	HardTL yaml.Node `yaml:"hardtl"`
}

func (self *Testcase) UnmarshalYAML(v *yaml.Node) error {
	var node testcaseNode
	err := v.Decode(&node)
	if err != nil {
		return err
	}

	var im testcase
	err = v.Decode(&im)
	if err != nil {
		return err
	}
	self.testcase = im

	if node.SoftTL.IsZero() {
		self.SoftTL = 25
	}
	if node.HardTL.IsZero() {
		self.HardTL = 30
	}

	return nil
}
