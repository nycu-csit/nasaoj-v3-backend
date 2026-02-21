package parsers

import (
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type toolInputArg struct {
	Name string                  `yaml:"name"`
	Type types.ToolInputTypeType `yaml:"-"`
}

func (self *toolInputArg) UnmarshalYAML(v *yaml.Node) error {
	var raw struct {
		Name string `yaml:"name"`
		Type string `yaml:"type"`
	}
	if err := v.Decode(&raw); err != nil {
		return err
	}

	self.Name = raw.Name

	switch raw.Type {
	case "", "string":
		self.Type = types.ToolInputTypeString
	case "numeric":
		self.Type = types.ToolInputTypeNumeric
	case "boolean":
		self.Type = types.ToolInputTypeBoolean
	default:
		return fmt.Errorf("unknown input arg type: %s", raw.Type)
	}

	return nil
}

type tool struct {
	ToolId int64 `yaml:"toolId"`

	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Testcase    Testcase          `yaml:"testcase"`
	Args        map[string]string `yaml:"args"`
	InputArgs   []toolInputArg    `yaml:"inputArgs"`
	Tag         string            `yaml:"tag"`
	PreScript   Content           `yaml:"preScript"`
	PostScript  Content           `yaml:"postScript"`
	Cooldown    int64             `yaml:"cooldown"`
}

type Tool struct {
	tool `yaml:",inline"`
}

type toolNode struct {
	Tag      yaml.Node `yaml:"tag"`
	Cooldown yaml.Node `yaml:"cooldown"`
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

	if node.Cooldown.IsZero() {
		self.tool.Cooldown = 300
	}

	return nil
}

func (self *Tool) ToModelTool(
	ctx context.Context,
	contentFiles map[string]models.ContentFile,
) (models.Tool, error) {
	inputArgs := make([]models.ToolInputArg, len(self.InputArgs))
	for i, arg := range self.InputArgs {
		inputArgs[i] = models.ToolInputArg{
			Name: arg.Name,
			Type: arg.Type, // defaults to types.ToolInputTypeString (0)
		}
	}

	ret := models.Tool{
		ToolId: self.ToolId,

		Name:        self.Name,
		Description: self.Description,
		Args:        self.Args,
		InputArgs:   inputArgs,
		Tag:         self.Tag,
		PreScript: models.Content{
			Scripts: self.PreScript.Scripts,
		},
		PostScript: models.Content{
			Scripts: self.PostScript.Scripts,
		},
		Cooldown: self.Cooldown,
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
