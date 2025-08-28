package parsers

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type tool struct {
	ToolId int64 `yaml:"toolId"`

	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Testcase    Testcase          `yaml:"testcase"`
	Args        map[string]string `yaml:"args"`
	InputArgs   []string          `yaml:"inputArgs"`
	Tag         string            `yaml:"tag"`
	PreScript   Content           `yaml:"preScript"`
	PostScript  Content           `yaml:"postScript"`
}

type Tool struct {
	tool `yaml:",inline"`
}

type toolNode struct {
	Tag yaml.Node `yaml:"tag"`
}

func (self *Tool) UnmarshalYAML(v *yaml.Node) error {
	var node problemNode
	err := v.Decode(&node)
	if err != nil {
		return err
	}

	var im tool
	if err = v.Decode(&im); err != nil {
		return err
	}
	self.tool = im

	if node.Tag.IsZero() {
		self.tool.Tag = "default"
	}

	return nil
}

func (self *Tool) ToModelTool(
	ctx context.Context,
	contentFiles map[string]models.ContentFile,
) (models.Tool, error) {
	ret := models.Tool{
		ToolId: self.ToolId,

		Name:        self.Name,
		Description: self.Description,
		Args:        self.Args,
		InputArgs:   self.InputArgs,
		Tag:         self.Tag,
		PreScript: models.Content{
			Scripts: self.PreScript.Scripts,
		},
		PostScript: models.Content{
			Scripts: self.PostScript.Scripts,
		},
		Timeout: self.Testcase.HardTL,
	}
	testcase, err := self.Testcase.ToModelTestcase(ctx, contentFiles)
	if err != nil {
		tracing.Logger.Ctx(ctx).
			Error("failed to extract testcase",
				zap.Error(err))
		return models.Tool{}, err
	}
	ret.Testcase = testcase
	return ret, nil
}
