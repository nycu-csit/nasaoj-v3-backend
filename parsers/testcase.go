package parsers

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/types"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Content struct {
	Files   []string `yaml:"files"`
	Scripts []string `yaml:"scripts"`
}

type testcase struct {
	Content `yaml:",inline"`

	Name          string            `yaml:"name"`
	Args          map[string]string `yaml:"args"`
	Timeout       int64             `yaml:"timeout"`
	ResourceGroup string            `yaml:"resourceGroup,omitempty`
}

type Testcase struct {
	testcase `yaml:",inline"`
}

type testcaseNode struct {
	Timeout yaml.Node `yaml:"timeout"`
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

	if node.Timeout.IsZero() {
		self.Timeout = 30
	}

	return nil
}

func (self *Content) ToModelContent(
	ctx context.Context,
	contentFiles map[string]models.ContentFile,
) (models.Content, error) {
	ret := models.Content{
		Scripts: self.Scripts,
	}

	for _, file := range self.Files {
		contentFile, ok := contentFiles[file]
		if !ok {
			tracing.Logger.Ctx(ctx).
				Error("file not exist",
					zap.String("file.path", file))
			return models.Content{},
				types.FileNotExistError
		}
		ret.Files = append(ret.Files, contentFile)
	}

	return ret, nil
}

func (self *Testcase) ToModelTestcase(
	ctx context.Context,
	contentFiles map[string]models.ContentFile,
) (models.Testcase, error) {
	ret := models.Testcase{
		Name:          self.Name,
		Args:          self.Args,
		Timeout:       self.Timeout,
		ResourceGroup: self.ResourceGroup,
	}

	if res, err := self.Content.ToModelContent(ctx, contentFiles); err == nil {
		ret.Content = res
	} else {
		tracing.Logger.Ctx(ctx).
			Error("failed to transform content",
				zap.Error(err))
		return models.Testcase{}, err
	}

	return ret, nil
}
