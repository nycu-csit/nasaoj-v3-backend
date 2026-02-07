package parsers

import (
	"context"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type ProblemIdType = int64
type TagType = string
type problem struct {
	ProblemId ProblemIdType `yaml:"problemId"`

	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	PdfUrl      string            `yaml:"pdfUrl"`
	Stages      map[string]Stage  `yaml:"stages"`
	Args        map[string]string `yaml:"args"`
	Tag         TagType           `yaml:"tag"`
	PreScript   Content           `yaml:"preScript"`
	PostScript  Content           `yaml:"postScript"`
	Deadline    int64             `yaml:"deadline"`
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
		self.problem.Tag = "default"
	}

	return nil
}

func (self *Problem) ToModelProblem(
	ctx context.Context,
	contentFiles map[string]models.ContentFile,
) (models.Problem, error) {
	ret := models.Problem{
		ProblemId: self.ProblemId,

		Name:        self.Name,
		Description: self.Description,
		PdfUrl:      self.PdfUrl,
		Stages:      make(map[string]models.Stage),
		Args:        self.Args,
		Tag:         self.Tag,
		Deadline:    self.Deadline,
	}

	if res, err := self.PreScript.ToModelContent(ctx, contentFiles); err == nil {
		ret.PreScript = res
	} else {
		tracing.Logger.Ctx(ctx).
			Error("failed to transform content",
				zap.Error(err))
		return models.Problem{}, err
	}

	if res, err := self.PostScript.ToModelContent(ctx, contentFiles); err == nil {
		ret.PostScript = res
	} else {
		tracing.Logger.Ctx(ctx).
			Error("failed to transform content",
				zap.Error(err))
		return models.Problem{}, err
	}

	ret.Timeout = 0
	for stageName, stage := range self.Stages {
		retStage := models.Stage{
			Before: stage.Before,
			Args:   stage.Args,
			Score:  stage.Score,
		}

		if res, err := stage.PreScript.
			ToModelContent(ctx, contentFiles); err == nil {
			retStage.PreScript = res
		} else {
			tracing.Logger.Ctx(ctx).
				Error("failed to transform content",
					zap.Error(err))
			return models.Problem{}, err
		}

		if res, err := stage.PostScript.
			ToModelContent(ctx, contentFiles); err == nil {
			retStage.PostScript = res
		} else {
			tracing.Logger.Ctx(ctx).
				Error("failed to transform content",
					zap.Error(err))
			return models.Problem{}, err
		}

		maxTL := int64(0)
		for _, testcase := range stage.Testcases {
			maxTL = max(maxTL, testcase.HardTL)
			retTestcase, err := testcase.ToModelTestcase(ctx, contentFiles)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to extract testcase",
						zap.Error(err))
				return models.Problem{}, err
			}
			retStage.Testcases = append(retStage.Testcases, retTestcase)
		}
		ret.Timeout += maxTL
		ret.Stages[stageName] = retStage
	}

	// Handle after dependencies
	for stageName, stage := range self.Stages {
		for _, prev := range stage.After {
			prevStage := ret.Stages[prev]
			prevStage.Before = append(prevStage.Before, stageName)
			ret.Stages[prev] = prevStage
		}
	}

	return ret, nil
}
