package parsers

import (
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gopkg.in/yaml.v3"
)

type Content struct {
	Files   []string `yaml:"files"`
	Scripts []string `yaml:"scripts"`
}

type testcase struct {
	Content `yaml:",inline"`

	Name   string            `yaml:"name"`
	Args   map[string]string `yaml:"args"`
	SoftTL int64             `yaml:"softTL"`
	HardTL int64             `yaml:"hardTL"`
}

type Testcase struct {
	testcase `yaml:",inline"`
}

type testcaseNode struct {
	SoftTL yaml.Node `yaml:"softTL"`
	HardTL yaml.Node `yaml:"hardTL"`
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

func (self *Testcase) ToModelTestcase(
	contentFiles map[string]models.ContentFile,
) (models.Testcase, error) {
	ret := models.Testcase{
		Name:   self.Name,
		Args:   self.Args,
		SoftTL: self.SoftTL,
		HardTL: self.HardTL,
		Content: models.Content{
			Scripts: self.Content.Scripts,
		},
	}

	for _, file := range self.Content.Files {
		contentFile, ok := contentFiles[file]
		if !ok {
			return ret, fmt.Errorf("file '%s' not exist", file)
		}
		ret.Content.Files = append(ret.Content.Files, contentFile)
	}
	return ret, nil
}
