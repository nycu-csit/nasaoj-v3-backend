package models

import "gopkg.in/yaml.v3"

type ProblemRepository = map[ProblemIdType]Problem

type ProblemIdType = int64
type TagType = string
type problem struct {
	Name        string            `json:"name" yaml:"name"`
	ProblemId   ProblemIdType     `json:"problemId" yaml:"problemId"`
	Description string            `json:"description" yaml:"description"`
	Stages      map[string]Stage  `json:"stages" yaml:"stages"`
	Args        map[string]string `json:"args" yaml:"args"`
	Tag         TagType           `json:"tag" yaml:"tag"`
	PreScript   Content           `json:"preScript" yaml:"preScript"`
	PostScript  Content           `json:"postScript" yaml:"postScript"`

	Timeout int64 `json:"-" yaml:"-"`
}

type Problem struct {
	problem `yaml:",inline"`
}

type problemNode struct {
	Tag yaml.Node `yaml:"tag"`
}

func (self *Problem) UnmarshalYAML(v *yaml.Node) error {
	var node problemNode
	err := v.Decode(&node)
	if err != nil {
		return err
	}

	var im problem
	err = v.Decode(&im)
	if err != nil {
		return err
	}
	self.problem = im

	if node.Tag.IsZero() {
		self.Tag = "default"
	}

	return nil
}
