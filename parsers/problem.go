package parsers

import (
	"context"
	"fmt"

	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/models"
	"gitlab.it.cs.nycu.edu.tw/nasa/nasaoj-v3/judge/tracing"
	"go.uber.org/zap"
	"go.yaml.in/yaml/v3"
)

type ProblemIdType = int64
type TagType = string

type ProblemSpec struct {
	Type string `yaml:"type"`
	Link string `yaml:"link"`
}

type problem struct {
	ProblemId ProblemIdType `yaml:"problemId"`

	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Spec        ProblemSpec       `yaml:"spec"`
	Stages      map[string]Stage  `yaml:"stages"`
	Args        map[string]string `yaml:"args"`
	Tag         TagType           `yaml:"tag"`
	PreScript   Content           `yaml:"preScript"`
	PostScript  Content           `yaml:"postScript"`
	Cooldown    int64             `yaml:"cooldown"`
	Deadline    int64             `yaml:"deadline"`
	ScoreType   string            `yaml:"scoreType"`
}

type Problem struct {
	problem `yaml:",inline"`
}

type problemNode struct {
	Tag      yaml.Node `yaml:"tag"`
	Cooldown yaml.Node `yaml:"cooldown"`
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

	if node.Cooldown.IsZero() {
		self.problem.Cooldown = 300
	}

	switch self.problem.Spec.Type {
	case "pdf", "site":
	case "":
		if self.problem.Spec.Link != "" {
			return fmt.Errorf("missing spec.type, must be 'pdf' or 'site'")
		}
	default:
		return fmt.Errorf("invalid spec.type: %q, must be 'pdf' or 'site'",
			self.problem.Spec.Type)
	}

	switch self.problem.ScoreType {
	case "", "default":
		self.problem.ScoreType = "default"
	case "self-managed":
	default:
		return fmt.Errorf("invalid scoreType: %q, must be 'default' or 'self-managed'",
			self.problem.ScoreType)
	}

	return nil
}

func (self *Problem) ToModelProblem(
	ctx context.Context,
	contentFiles map[string]models.ContentFile,
) (models.Problem, error) {
	var scoreType models.ScoreTypeType
	switch self.ScoreType {
	case "self-managed":
		scoreType = models.ScoreTypeSelfManaged
	default:
		scoreType = models.ScoreTypeDefault
	}

	ret := models.Problem{
		ProblemId: self.ProblemId,

		Name:        self.Name,
		Description: self.Description,
		Spec: models.ProblemSpec{
			Type: self.Spec.Type,
			Link: self.Spec.Link,
		},
		Stages:    make(map[string]models.Stage),
		Args:      self.Args,
		Tag:       self.Tag,
		Deadline:  self.Deadline,
		Cooldown:  self.Cooldown,
		ScoreType: scoreType,
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

		for _, testcase := range stage.Testcases {
			retTestcase, err := testcase.ToModelTestcase(ctx, contentFiles)
			if err != nil {
				tracing.Logger.Ctx(ctx).
					Error("failed to extract testcase",
						zap.Error(err))
				return models.Problem{}, err
			}
			retStage.Testcases = append(retStage.Testcases, retTestcase)
		}
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
